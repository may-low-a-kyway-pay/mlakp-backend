package dashboard

import (
	"context"
	"errors"
	"testing"
)

func TestServiceGetValidatesAndTrimsInput(t *testing.T) {
	wantSnapshot := Snapshot{
		Totals: Totals{
			YouOwe: DashboardAmount{AmountMinor: 1250, DebtCount: 1},
		},
	}

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
			store := &fakeStore{snapshot: wantSnapshot}
			service := NewService(store)
			got, err := service.Get(context.Background(), tt.userID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Get() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && store.userID != "user-1" {
				t.Fatalf("userID = %q, want user-1", store.userID)
			}
			if tt.wantErr == nil && got.Totals.YouOwe.AmountMinor != wantSnapshot.Totals.YouOwe.AmountMinor {
				t.Fatalf("snapshot.YouOwe.AmountMinor = %d, want %d", got.Totals.YouOwe.AmountMinor, wantSnapshot.Totals.YouOwe.AmountMinor)
			}
		})
	}
}

type fakeStore struct {
	userID   string
	snapshot Snapshot
}

func (s *fakeStore) GetSnapshot(_ context.Context, userID string) (Snapshot, error) {
	s.userID = userID
	return s.snapshot, nil
}
