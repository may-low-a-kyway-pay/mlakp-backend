package dashboard

import (
	"context"
	"errors"
	"testing"
)

func TestServiceGetValidatesAndTrimsInput(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		wantErr error
	}{
		{name: "missing user", userID: " ", wantErr: ErrInvalidUserID},
		{name: "valid", userID: " user-1 "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{}
			service := NewService(store)
			_, err := service.Get(context.Background(), tt.userID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Get() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && store.userID != "user-1" {
				t.Fatalf("userID = %q, want user-1", store.userID)
			}
		})
	}
}

type fakeStore struct {
	userID string
}

func (s *fakeStore) GetTotals(_ context.Context, userID string) (Totals, error) {
	s.userID = userID
	return Totals{}, nil
}
