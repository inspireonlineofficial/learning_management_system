package auth

import (
	"context"
	domainauth "lms-backend/internal/domain/auth"
	"testing"

	"github.com/google/uuid"
)

func TestRoleForGoogleOAuthEmailPromotesOfficialAccount(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		email    string
		want     string
	}{
		{
			name:     "official google account",
			provider: "google",
			email:    "inspireonlineofficial@gmail.com",
			want:     "admin",
		},
		{
			name:     "official google account case insensitive",
			provider: "google",
			email:    " InspireOnlineOfficial@gmail.com ",
			want:     "admin",
		},
		{
			name:     "same email on another provider is not admin",
			provider: "github",
			email:    "inspireonlineofficial@gmail.com",
			want:     "student",
		},
		{
			name:     "regular google account",
			provider: "google",
			email:    "student@example.com",
			want:     "student",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := roleForGoogleOAuthEmail(tt.provider, tt.email); got != tt.want {
				t.Fatalf("roleForGoogleOAuthEmail() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEnsureGoogleOAuthAdminRolePromotesExistingOfficialAccount(t *testing.T) {
	repo := newMockUserRepo()
	user := &domainauth.User{
		ID:              uuid.New(),
		FullName:        "Inspire Official",
		Email:           "inspireonlineofficial@gmail.com",
		Role:            "student",
		Status:          "active",
		ProfileComplete: false,
	}
	repo.users[user.Email] = user

	service := &authService{deps: ServiceDeps{UserRepo: repo}}
	if err := service.ensureGoogleOAuthAdminRole(context.Background(), "google", user); err != nil {
		t.Fatalf("ensureGoogleOAuthAdminRole() error = %v", err)
	}

	updated, err := repo.FindByEmail(context.Background(), user.Email)
	if err != nil {
		t.Fatalf("expected user in repo: %v", err)
	}
	if updated.Role != "admin" {
		t.Fatalf("role = %q, want admin", updated.Role)
	}
	if !updated.ProfileComplete {
		t.Fatal("profile should be complete for the official admin account")
	}
}

func TestEnsureGoogleOAuthAdminRoleIgnoresNonGoogleProvider(t *testing.T) {
	user := &domainauth.User{
		ID:       uuid.New(),
		Email:    "inspireonlineofficial@gmail.com",
		Role:     "student",
		Status:   "active",
		FullName: "Inspire Official",
	}

	service := &authService{deps: ServiceDeps{UserRepo: newMockUserRepo()}}
	if err := service.ensureGoogleOAuthAdminRole(context.Background(), "github", user); err != nil {
		t.Fatalf("ensureGoogleOAuthAdminRole() error = %v", err)
	}
	if user.Role != "student" {
		t.Fatalf("role = %q, want student", user.Role)
	}
}
