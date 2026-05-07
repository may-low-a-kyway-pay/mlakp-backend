package handlers

import (
	"testing"
	"time"

	"mlakp-backend/internal/groups"
)

func TestToGroupMemberResponseIncludesUserSummary(t *testing.T) {
	joinedAt := time.Date(2026, 5, 6, 9, 30, 0, 0, time.UTC)

	got := toGroupMemberResponse(groups.Member{
		ID:       "member-1",
		GroupID:  "group-1",
		UserID:   "user-1",
		Role:     groups.RoleOwner,
		JoinedAt: joinedAt,
		User: &groups.MemberUser{
			ID:    "user-1",
			Name:  "Thomas",
			Email: "thomas@example.com",
		},
	})

	if got.User == nil {
		t.Fatal("got.User = nil, want user summary")
	}
	if got.User.ID != "user-1" || got.User.Name != "Thomas" || got.User.Email != "thomas@example.com" {
		t.Fatalf("got.User = %+v, want user summary", got.User)
	}
	if got.JoinedAt != "2026-05-06T09:30:00Z" {
		t.Fatalf("got.JoinedAt = %q, want RFC3339 UTC timestamp", got.JoinedAt)
	}
}

func TestToGroupMemberResponseOmitsMissingUserSummary(t *testing.T) {
	got := toGroupMemberResponse(groups.Member{
		ID:      "member-1",
		GroupID: "group-1",
		UserID:  "user-1",
		Role:    groups.RoleMember,
	})

	if got.User != nil {
		t.Fatalf("got.User = %+v, want nil", got.User)
	}
}
