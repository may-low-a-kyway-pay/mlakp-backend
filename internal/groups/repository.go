package groups

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"mlakp-backend/internal/postgres/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewRepository(pool *pgxpool.Pool, queries *sqlc.Queries) *Repository {
	return &Repository{
		pool:    pool,
		queries: queries,
	}
}

func (r *Repository) Create(ctx context.Context, name, createdBy string) (Group, error) {
	createdByUUID, err := parseUUID(createdBy)
	if err != nil {
		return Group{}, ErrInvalidUserID
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Group{}, err
	}
	defer rollbackUnlessCommitted(ctx, tx)

	qtx := r.queries.WithTx(tx)
	group, err := qtx.CreateGroup(ctx, sqlc.CreateGroupParams{
		Name:      name,
		CreatedBy: createdByUUID,
	})
	if err != nil {
		return Group{}, err
	}

	// Group creation and initial ownership must commit or roll back together.
	if _, err := qtx.CreateGroupMember(ctx, sqlc.CreateGroupMemberParams{
		GroupID: group.ID,
		UserID:  createdByUUID,
		Role:    RoleOwner,
	}); err != nil {
		return Group{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Group{}, err
	}

	return groupFromSQLC(group), nil
}

func (r *Repository) ListForUser(ctx context.Context, userID string) ([]Group, error) {
	userUUID, err := parseUUID(userID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	rows, err := r.queries.ListGroupsForUser(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	groups := make([]Group, 0, len(rows))
	for _, row := range rows {
		groups = append(groups, groupFromSQLC(row))
	}

	return groups, nil
}

func (r *Repository) GetForUser(ctx context.Context, groupID, userID string) (GroupDetails, error) {
	groupUUID, err := parseUUID(groupID)
	if err != nil {
		return GroupDetails{}, ErrInvalidGroupID
	}
	userUUID, err := parseUUID(userID)
	if err != nil {
		return GroupDetails{}, ErrInvalidUserID
	}

	group, err := r.queries.GetGroupForUser(ctx, sqlc.GetGroupForUserParams{
		ID:     groupUUID,
		UserID: userUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GroupDetails{}, ErrNotFound
		}
		return GroupDetails{}, err
	}

	memberRows, err := r.queries.ListGroupMembersForUser(ctx, sqlc.ListGroupMembersForUserParams{
		GroupID: groupUUID,
		UserID:  userUUID,
	})
	if err != nil {
		return GroupDetails{}, err
	}

	members := make([]Member, 0, len(memberRows))
	for _, row := range memberRows {
		members = append(members, memberFromListRow(row))
	}

	return GroupDetails{
		Group:   groupFromSQLC(group),
		Members: members,
	}, nil
}

func (r *Repository) AddMember(ctx context.Context, groupID, ownerID, memberUserID string) (Member, error) {
	groupUUID, err := parseUUID(groupID)
	if err != nil {
		return Member{}, ErrInvalidGroupID
	}
	ownerUUID, err := parseUUID(ownerID)
	if err != nil {
		return Member{}, ErrInvalidUserID
	}
	memberUUID, err := parseUUID(memberUserID)
	if err != nil {
		return Member{}, ErrInvalidUserID
	}

	// Ownership is checked in the repository so callers cannot bypass it.
	isOwner, err := r.queries.IsGroupOwner(ctx, sqlc.IsGroupOwnerParams{
		GroupID: groupUUID,
		UserID:  ownerUUID,
	})
	if err != nil {
		return Member{}, err
	}
	if !isOwner {
		return Member{}, ErrForbidden
	}

	userExists, err := r.queries.UserExists(ctx, memberUUID)
	if err != nil {
		return Member{}, err
	}
	if !userExists {
		return Member{}, ErrMemberNotFound
	}

	member, err := r.queries.CreateGroupMember(ctx, sqlc.CreateGroupMemberParams{
		GroupID: groupUUID,
		UserID:  memberUUID,
		Role:    RoleMember,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Member{}, ErrMemberConflict
		}
		return Member{}, err
	}

	return memberFromSQLC(member), nil
}

func groupFromSQLC(group sqlc.Group) Group {
	return Group{
		ID:        formatUUID(group.ID),
		Name:      group.Name,
		CreatedBy: formatUUID(group.CreatedBy),
		CreatedAt: group.CreatedAt.Time,
		UpdatedAt: group.UpdatedAt.Time,
	}
}

func memberFromSQLC(member sqlc.GroupMember) Member {
	return Member{
		ID:       formatUUID(member.ID),
		GroupID:  formatUUID(member.GroupID),
		UserID:   formatUUID(member.UserID),
		Role:     member.Role,
		JoinedAt: member.JoinedAt.Time,
	}
}

func memberFromListRow(member sqlc.ListGroupMembersForUserRow) Member {
	userID := formatUUID(member.UserID)

	return Member{
		ID:       formatUUID(member.ID),
		GroupID:  formatUUID(member.GroupID),
		UserID:   userID,
		Role:     member.Role,
		JoinedAt: member.JoinedAt.Time,
		User: &MemberUser{
			ID:       userID,
			Name:     member.UserName,
			Username: member.UserUsername,
			Email:    member.UserEmail,
		},
	}
}

func parseUUID(value string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	if err := uuid.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	if !uuid.Valid {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid")
	}

	return uuid, nil
}

func formatUUID(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}

	encoded := hex.EncodeToString(uuid.Bytes[:])
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
}

func rollbackUnlessCommitted(ctx context.Context, tx pgx.Tx) {
	// pgx returns ErrTxClosed after Commit; ignoring rollback keeps defer cleanup simple.
	_ = tx.Rollback(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
