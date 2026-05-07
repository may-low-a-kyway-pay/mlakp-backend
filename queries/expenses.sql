-- name: IsGroupMember :one
SELECT EXISTS (
    SELECT 1
    FROM group_members
    WHERE group_id = $1
      AND user_id = $2
);

-- name: CreateExpense :one
INSERT INTO expenses (
    group_id,
    title,
    description,
    total_amount_minor,
    currency,
    paid_by,
    split_type,
    receipt_url,
    expense_date,
    created_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING id, group_id, title, description, total_amount_minor, currency, paid_by, split_type, receipt_url, expense_date, created_by, created_at, updated_at;

-- name: CreateExpenseParticipant :one
INSERT INTO expense_participants (expense_id, user_id, share_amount_minor)
VALUES ($1, $2, $3)
RETURNING id, expense_id, user_id, share_amount_minor, created_at;

-- name: CreateDebt :one
INSERT INTO debts (
    expense_id,
    debtor_id,
    creditor_id,
    original_amount_minor,
    remaining_amount_minor
)
VALUES ($1, $2, $3, $4, $4)
RETURNING id, expense_id, debtor_id, creditor_id, original_amount_minor, remaining_amount_minor, status, accepted_at, rejected_at, settled_at, created_at, updated_at;

-- name: GetExpenseForUser :one
SELECT e.id, e.group_id, e.title, e.description, e.total_amount_minor, e.currency, e.paid_by, e.split_type, e.receipt_url, e.expense_date, e.created_by, e.created_at, e.updated_at
FROM expenses e
WHERE e.id = $1
  AND EXISTS (
      SELECT 1
      FROM group_members gm
      WHERE gm.group_id = e.group_id
        AND gm.user_id = $2
  );

-- name: ListExpenseParticipants :many
SELECT id, expense_id, user_id, share_amount_minor, created_at
FROM expense_participants
WHERE expense_id = $1
ORDER BY created_at ASC, id ASC;

-- name: ListDebtsByExpense :many
SELECT id, expense_id, debtor_id, creditor_id, original_amount_minor, remaining_amount_minor, status, accepted_at, rejected_at, settled_at, created_at, updated_at
FROM debts
WHERE expense_id = $1
ORDER BY created_at ASC, id ASC;

-- name: ListGroupExpensesForUser :many
SELECT e.id, e.group_id, e.title, e.description, e.total_amount_minor, e.currency, e.paid_by, e.split_type, e.receipt_url, e.expense_date, e.created_by, e.created_at, e.updated_at
FROM expenses e
WHERE e.group_id = $1
  AND EXISTS (
      SELECT 1
      FROM group_members gm
      WHERE gm.group_id = e.group_id
        AND gm.user_id = $2
  )
ORDER BY e.expense_date DESC NULLS LAST, e.created_at DESC, e.id DESC;

-- name: GetDebtByID :one
SELECT id, expense_id, debtor_id, creditor_id, original_amount_minor, remaining_amount_minor, status, accepted_at, rejected_at, settled_at, created_at, updated_at
FROM debts
WHERE id = $1;

-- name: ListDebtsForUser :many
SELECT
    d.id,
    d.expense_id,
    e.title AS expense_title,
    d.debtor_id,
    debtor.name AS debtor_name,
    d.creditor_id,
    creditor.name AS creditor_name,
    d.original_amount_minor,
    d.remaining_amount_minor,
    d.status,
    d.accepted_at,
    d.rejected_at,
    d.settled_at,
    d.created_at,
    d.updated_at
FROM debts d
JOIN expenses e ON e.id = d.expense_id
JOIN users debtor ON debtor.id = d.debtor_id
JOIN users creditor ON creditor.id = d.creditor_id
WHERE (d.debtor_id = $1 OR d.creditor_id = $1)
  AND (sqlc.narg(status)::text IS NULL OR d.status = sqlc.narg(status)::text)
  AND (
      sqlc.narg(balance_type)::text IS NULL
      OR (sqlc.narg(balance_type)::text = 'owed' AND d.debtor_id = $1)
      OR (sqlc.narg(balance_type)::text = 'receivable' AND d.creditor_id = $1)
  )
ORDER BY d.updated_at DESC, d.created_at DESC, d.id DESC;

-- name: AcceptDebt :one
UPDATE debts
SET status = 'accepted',
    accepted_at = now()
WHERE id = $1
  AND debtor_id = $2
  AND status = 'pending'
RETURNING id, expense_id, debtor_id, creditor_id, original_amount_minor, remaining_amount_minor, status, accepted_at, rejected_at, settled_at, created_at, updated_at;

-- name: RejectDebt :one
UPDATE debts
SET status = 'rejected',
    rejected_at = now()
WHERE id = $1
  AND debtor_id = $2
  AND status = 'pending'
RETURNING id, expense_id, debtor_id, creditor_id, original_amount_minor, remaining_amount_minor, status, accepted_at, rejected_at, settled_at, created_at, updated_at;

-- name: ReviewRejectedDebt :one
UPDATE debts AS d
SET original_amount_minor = CASE WHEN sqlc.arg(has_adjusted_amount)::boolean THEN sqlc.arg(amount_minor)::bigint ELSE d.original_amount_minor END,
    remaining_amount_minor = CASE WHEN sqlc.arg(has_adjusted_amount)::boolean THEN sqlc.arg(amount_minor)::bigint ELSE d.original_amount_minor END,
    status = 'pending',
    accepted_at = NULL,
    rejected_at = NULL,
    settled_at = NULL
FROM expenses e
WHERE d.id = $1
  AND d.expense_id = e.id
  AND d.status = 'rejected'
  AND EXISTS (
      SELECT 1
      FROM group_members gm
      WHERE gm.group_id = e.group_id
        AND gm.user_id = $2
        AND gm.role = 'owner'
  )
RETURNING d.id, d.expense_id, d.debtor_id, d.creditor_id, d.original_amount_minor, d.remaining_amount_minor, d.status, d.accepted_at, d.rejected_at, d.settled_at, d.created_at, d.updated_at;

-- name: GetDebtReviewContext :one
SELECT d.status,
       EXISTS (
           SELECT 1
           FROM expenses e
           JOIN group_members gm ON gm.group_id = e.group_id
           WHERE e.id = d.expense_id
             AND gm.user_id = $2
             AND gm.role = 'owner'
       ) AS is_group_owner
FROM debts d
WHERE d.id = $1;
