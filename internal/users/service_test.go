package users

import (
	"context"
	"testing"
)

func TestServiceRegisterNormalizesEmailAndHashesPassword(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store, fakeHasher{})

	user, err := service.Register(context.Background(), " Thomas ", " THOMAS@Example.COM ", "password123")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if user.Email != "thomas@example.com" {
		t.Fatalf("user.Email = %q, want normalized email", user.Email)
	}
	if store.createdName != "Thomas" {
		t.Fatalf("createdName = %q, want trimmed name", store.createdName)
	}
	if store.createdPasswordHash != "hash:password123" {
		t.Fatalf("createdPasswordHash = %q, want hashed password", store.createdPasswordHash)
	}
}

func TestServiceRegisterRejectsInvalidInput(t *testing.T) {
	service := NewService(&fakeStore{}, fakeHasher{})

	tests := []struct {
		name     string
		userName string
		email    string
		password string
		wantErr  error
	}{
		{name: "empty name", userName: " ", email: "user@example.com", password: "password123", wantErr: ErrInvalidName},
		{name: "invalid email", userName: "User", email: "invalid", password: "password123", wantErr: ErrInvalidEmail},
		{name: "short password", userName: "User", email: "user@example.com", password: "short", wantErr: ErrInvalidPassword},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Register(context.Background(), tt.userName, tt.email, tt.password)
			if err != tt.wantErr {
				t.Fatalf("Register() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceAuthenticateNormalizesEmail(t *testing.T) {
	store := &fakeStore{
		user: PrivateUser{
			User: User{
				ID:    "user-1",
				Name:  "Thomas",
				Email: "thomas@example.com",
			},
			PasswordHash: "hash:password123",
		},
	}
	service := NewService(store, fakeHasher{})

	_, err := service.Authenticate(context.Background(), " THOMAS@Example.COM ", "password123")
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if store.lookupEmail != "thomas@example.com" {
		t.Fatalf("lookupEmail = %q, want normalized email", store.lookupEmail)
	}
}

type fakeStore struct {
	createdName         string
	createdEmail        string
	createdPasswordHash string
	lookupEmail         string
	user                PrivateUser
}

func (s *fakeStore) Create(_ context.Context, name, email, passwordHash string) (PrivateUser, error) {
	s.createdName = name
	s.createdEmail = email
	s.createdPasswordHash = passwordHash

	return PrivateUser{
		User: User{
			ID:    "user-1",
			Name:  name,
			Email: email,
		},
		PasswordHash: passwordHash,
	}, nil
}

func (s *fakeStore) GetByEmail(_ context.Context, email string) (PrivateUser, error) {
	s.lookupEmail = email
	return s.user, nil
}

func (s *fakeStore) GetByID(_ context.Context, id string) (User, error) {
	return User{ID: id}, nil
}

type fakeHasher struct{}

func (fakeHasher) HashPassword(password string) (string, error) {
	return "hash:" + password, nil
}

func (fakeHasher) ComparePassword(hash, password string) bool {
	return hash == "hash:"+password
}
