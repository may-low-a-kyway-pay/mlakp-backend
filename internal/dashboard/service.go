package dashboard

import (
	"context"
	"errors"
	"strings"
)

var ErrInvalidUserID = errors.New("user id is invalid")

type Store interface {
	GetTotals(ctx context.Context, userID string) (Totals, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Get(ctx context.Context, userID string) (Totals, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Totals{}, ErrInvalidUserID
	}

	return s.store.GetTotals(ctx, userID)
}
