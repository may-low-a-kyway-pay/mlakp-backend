package groups

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidName    = errors.New("group name must be between 1 and 120 characters")
	ErrInvalidUserID  = errors.New("user id is invalid")
	ErrInvalidGroupID = errors.New("group id is invalid")
	ErrNotFound       = errors.New("group not found")
	ErrForbidden      = errors.New("group action is forbidden")
	ErrMemberNotFound = errors.New("group member user not found")
	ErrMemberConflict = errors.New("user is already a group member")
)

type Store interface {
	Create(ctx context.Context, name, createdBy string) (Group, error)
	ListForUser(ctx context.Context, userID string) ([]Group, error)
	GetForUser(ctx context.Context, groupID, userID string) (GroupDetails, error)
	AddMember(ctx context.Context, groupID, ownerID, memberUserID string) (Member, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Create(ctx context.Context, name, createdBy string) (Group, error) {
	name = strings.TrimSpace(name)
	createdBy = strings.TrimSpace(createdBy)

	if err := validateName(name); err != nil {
		return Group{}, err
	}
	if createdBy == "" {
		return Group{}, ErrInvalidUserID
	}

	return s.store.Create(ctx, name, createdBy)
}

func (s *Service) ListForUser(ctx context.Context, userID string) ([]Group, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidUserID
	}

	return s.store.ListForUser(ctx, userID)
}

func (s *Service) GetForUser(ctx context.Context, groupID, userID string) (GroupDetails, error) {
	groupID = strings.TrimSpace(groupID)
	userID = strings.TrimSpace(userID)
	if groupID == "" {
		return GroupDetails{}, ErrInvalidGroupID
	}
	if userID == "" {
		return GroupDetails{}, ErrInvalidUserID
	}

	return s.store.GetForUser(ctx, groupID, userID)
}

func (s *Service) AddMember(ctx context.Context, groupID, ownerID, memberUserID string) (Member, error) {
	groupID = strings.TrimSpace(groupID)
	ownerID = strings.TrimSpace(ownerID)
	memberUserID = strings.TrimSpace(memberUserID)
	if groupID == "" {
		return Member{}, ErrInvalidGroupID
	}
	if ownerID == "" || memberUserID == "" {
		return Member{}, ErrInvalidUserID
	}

	return s.store.AddMember(ctx, groupID, ownerID, memberUserID)
}

func validateName(name string) error {
	if len(name) < 1 || len(name) > 120 {
		return ErrInvalidName
	}

	return nil
}
