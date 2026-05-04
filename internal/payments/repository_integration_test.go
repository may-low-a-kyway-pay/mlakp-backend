package payments

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"mlakp-backend/internal/postgres/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryConfirmConcurrentPayments(t *testing.T) {
	pool := newIntegrationPool(t)
	repository := NewRepository(pool, sqlc.New(pool))
	fixture := seedAcceptedDebt(t, pool, 1000)

	paymentOneID := seedPayment(t, pool, fixture.debtID, fixture.debtorID, fixture.creditorID, 700)
	paymentTwoID := seedPayment(t, pool, fixture.debtID, fixture.debtorID, fixture.creditorID, 700)

	type result struct {
		payment Payment
		err     error
	}
	results := make(chan result, 2)

	var wg sync.WaitGroup
	for _, paymentID := range []string{paymentOneID, paymentTwoID} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payment, err := repository.Confirm(context.Background(), paymentID, fixture.creditorID)
			results <- result{payment: payment, err: err}
		}()
	}
	wg.Wait()
	close(results)

	var confirmed int
	var exceeded int
	for result := range results {
		switch {
		case result.err == nil:
			confirmed++
			if result.payment.Status != StatusConfirmed {
				t.Fatalf("confirmed payment status = %q, want %q", result.payment.Status, StatusConfirmed)
			}
		case errors.Is(result.err, ErrAmountExceedsRemaining):
			exceeded++
		default:
			t.Fatalf("Confirm() error = %v, want nil or %v", result.err, ErrAmountExceedsRemaining)
		}
	}

	if confirmed != 1 || exceeded != 1 {
		t.Fatalf("confirmed=%d exceeded=%d, want confirmed=1 exceeded=1", confirmed, exceeded)
	}

	var remaining int64
	var debtStatus string
	err := pool.QueryRow(context.Background(), `SELECT remaining_amount_minor, status FROM debts WHERE id = $1`, fixture.debtID).Scan(&remaining, &debtStatus)
	if err != nil {
		t.Fatalf("query debt after concurrent confirms: %v", err)
	}
	if remaining != 300 {
		t.Fatalf("remaining_amount_minor = %d, want 300", remaining)
	}
	if debtStatus != DebtStatusPartiallySettled {
		t.Fatalf("debt status = %q, want %q", debtStatus, DebtStatusPartiallySettled)
	}
}

func TestRepositoryMarkConcurrentPaymentsRespectsPendingAmount(t *testing.T) {
	pool := newIntegrationPool(t)
	repository := NewRepository(pool, sqlc.New(pool))
	fixture := seedAcceptedDebt(t, pool, 1000)

	type result struct {
		payment Payment
		err     error
	}
	results := make(chan result, 2)

	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payment, err := repository.Mark(context.Background(), markParams{
				DebtID:      fixture.debtID,
				UserID:      fixture.debtorID,
				AmountMinor: 700,
			})
			results <- result{payment: payment, err: err}
		}()
	}
	wg.Wait()
	close(results)

	var marked int
	var exceeded int
	for result := range results {
		switch {
		case result.err == nil:
			marked++
			if result.payment.Status != StatusPendingConfirmation {
				t.Fatalf("marked payment status = %q, want %q", result.payment.Status, StatusPendingConfirmation)
			}
		case errors.Is(result.err, ErrAmountExceedsRemaining):
			exceeded++
		default:
			t.Fatalf("Mark() error = %v, want nil or %v", result.err, ErrAmountExceedsRemaining)
		}
	}

	if marked != 1 || exceeded != 1 {
		t.Fatalf("marked=%d exceeded=%d, want marked=1 exceeded=1", marked, exceeded)
	}

	var pendingTotal int64
	err := pool.QueryRow(context.Background(), `
		SELECT COALESCE(SUM(amount_minor), 0)::bigint
		FROM payments
		WHERE debt_id = $1
		  AND status = 'pending_confirmation'
	`, fixture.debtID).Scan(&pendingTotal)
	if err != nil {
		t.Fatalf("query pending payments: %v", err)
	}
	if pendingTotal != 700 {
		t.Fatalf("pending total = %d, want 700", pendingTotal)
	}
}

type paymentFixture struct {
	debtorID   string
	creditorID string
	debtID     string
}

func newIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := strings.TrimSpace(os.Getenv("MLAKP_TEST_DATABASE_URL"))
	if databaseURL == "" {
		databaseURL = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if databaseURL == "" {
		t.Skip("set MLAKP_TEST_DATABASE_URL or DATABASE_URL to run PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}

	if _, err := adminPool.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_, _ = adminPool.Exec(cleanupCtx, `DROP SCHEMA IF EXISTS `+schema+` CASCADE`)
		adminPool.Close()
	})

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database url: %v", err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema + ",public"

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatalf("open schema pool: %v", err)
	}
	t.Cleanup(pool.Close)

	applyMigrations(t, pool)
	return pool
}

func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	root := repoRoot(t)
	matches, err := filepath.Glob(filepath.Join(root, "migrations", "*.up.sql"))
	if err != nil {
		t.Fatalf("list migrations: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("no up migrations found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for _, path := range matches {
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := pool.Exec(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("apply migration %s: %v", filepath.Base(path), err)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("could not find repo root")
		}
		wd = parent
	}
}

func seedAcceptedDebt(t *testing.T, pool *pgxpool.Pool, amountMinor int64) paymentFixture {
	t.Helper()

	ctx := context.Background()
	var creditorID string
	var debtorID string
	var groupID string
	var expenseID string
	var debtID string

	if err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash)
		VALUES ('Creditor', 'creditor-' || gen_random_uuid() || '@example.com', 'hash')
		RETURNING id::text
	`).Scan(&creditorID); err != nil {
		t.Fatalf("seed creditor: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash)
		VALUES ('Debtor', 'debtor-' || gen_random_uuid() || '@example.com', 'hash')
		RETURNING id::text
	`).Scan(&debtorID); err != nil {
		t.Fatalf("seed debtor: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO groups (name, created_by)
		VALUES ('Home', $1)
		RETURNING id::text
	`, creditorID).Scan(&groupID); err != nil {
		t.Fatalf("seed group: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO group_members (group_id, user_id, role)
		VALUES ($1, $2, 'owner'), ($1, $3, 'member')
	`, groupID, creditorID, debtorID); err != nil {
		t.Fatalf("seed group members: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO expenses (group_id, title, total_amount_minor, currency, paid_by, split_type, created_by)
		VALUES ($1, 'Dinner', $2, 'THB', $3, 'manual', $3)
		RETURNING id::text
	`, groupID, amountMinor, creditorID).Scan(&expenseID); err != nil {
		t.Fatalf("seed expense: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO expense_participants (expense_id, user_id, share_amount_minor)
		VALUES ($1, $2, $4), ($1, $3, $4)
	`, expenseID, creditorID, debtorID, amountMinor); err != nil {
		t.Fatalf("seed participants: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO debts (expense_id, debtor_id, creditor_id, original_amount_minor, remaining_amount_minor, status, accepted_at)
		VALUES ($1, $2, $3, $4, $4, 'accepted', now())
		RETURNING id::text
	`, expenseID, debtorID, creditorID, amountMinor).Scan(&debtID); err != nil {
		t.Fatalf("seed debt: %v", err)
	}

	return paymentFixture{
		debtorID:   debtorID,
		creditorID: creditorID,
		debtID:     debtID,
	}
}

func seedPayment(t *testing.T, pool *pgxpool.Pool, debtID, paidBy, receivedBy string, amountMinor int64) string {
	t.Helper()

	var paymentID string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO payments (debt_id, paid_by, received_by, amount_minor)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, debtID, paidBy, receivedBy, amountMinor).Scan(&paymentID); err != nil {
		t.Fatalf("seed payment: %v", err)
	}

	return paymentID
}
