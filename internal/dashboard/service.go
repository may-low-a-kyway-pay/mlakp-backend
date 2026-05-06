package dashboard

import (
	"context"
	"errors"
	"strings"
)

var ErrInvalidUserID = errors.New("user id is invalid")

type Store interface {
	GetSnapshot(ctx context.Context, userID string) (Snapshot, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Get(ctx context.Context, userID string) (Snapshot, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Snapshot{}, ErrInvalidUserID
	}

	return s.store.GetSnapshot(ctx, userID)
}
