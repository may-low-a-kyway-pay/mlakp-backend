package groups

import (
	"context"
	"testing"
)

func TestServiceCreateTrimsName(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)

	group, err := service.Create(context.Background(), " Home ", "user-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if group.Name != "Home" {
		t.Fatalf("group.Name = %q, want Home", group.Name)
	}
	if store.createdName != "Home" {
		t.Fatalf("createdName = %q, want trimmed name", store.createdName)
	}
	if store.createdBy != "user-1" {
		t.Fatalf("createdBy = %q, want user-1", store.createdBy)
	}
}

func TestServiceCreateRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeStore{})

	tests := []struct {
		name      string
		groupName string
		createdBy string
		wantErr   error
	}{
		{name: "empty name", groupName: " ", createdBy: "user-1", wantErr: ErrInvalidName},
		{name: "empty user", groupName: "Home", createdBy: " ", wantErr: ErrInvalidUserID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), tt.groupName, tt.createdBy)
			if err != tt.wantErr {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceAddMemberValidatesIDs(t *testing.T) {
	service := NewService(&fakeStore{})

	tests := []struct {
		name         string
		groupID      string
		ownerID      string
		memberUserID string
		wantErr      error
	}{
		{name: "empty group", groupID: " ", ownerID: "owner-1", memberUserID: "user-2", wantErr: ErrInvalidGroupID},
		{name: "empty owner", groupID: "group-1", ownerID: " ", memberUserID: "user-2", wantErr: ErrInvalidUserID},
		{name: "empty member", groupID: "group-1", ownerID: "owner-1", memberUserID: " ", wantErr: ErrInvalidUserID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.AddMember(context.Background(), tt.groupID, tt.ownerID, tt.memberUserID)
			if err != tt.wantErr {
				t.Fatalf("AddMember() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceListAndGetValidateUser(t *testing.T) {
	service := NewService(&fakeStore{})

	if _, err := service.ListForUser(context.Background(), " "); err != ErrInvalidUserID {
		t.Fatalf("ListForUser() error = %v, want %v", err, ErrInvalidUserID)
	}
	if _, err := service.GetForUser(context.Background(), "group-1", " "); err != ErrInvalidUserID {
		t.Fatalf("GetForUser() error = %v, want %v", err, ErrInvalidUserID)
	}
	if _, err := service.GetForUser(context.Background(), " ", "user-1"); err != ErrInvalidGroupID {
		t.Fatalf("GetForUser() error = %v, want %v", err, ErrInvalidGroupID)
	}
}

type fakeStore struct {
	createdName string
	createdBy   string
}

func (s *fakeStore) Create(_ context.Context, name, createdBy string) (Group, error) {
	s.createdName = name
	s.createdBy = createdBy
	return Group{
		ID:        "group-1",
		Name:      name,
		CreatedBy: createdBy,
	}, nil
}

func (s *fakeStore) ListForUser(_ context.Context, _ string) ([]Group, error) {
	return []Group{{ID: "group-1"}}, nil
}

func (s *fakeStore) GetForUser(_ context.Context, _, _ string) (GroupDetails, error) {
	return GroupDetails{Group: Group{ID: "group-1"}}, nil
}

func (s *fakeStore) AddMember(_ context.Context, groupID, _, memberUserID string) (Member, error) {
	return Member{
		ID:      "member-1",
		GroupID: groupID,
		UserID:  memberUserID,
		Role:    RoleMember,
	}, nil
}
