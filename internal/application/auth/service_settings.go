package auth

import (
	"context"
	domainauth "lms-backend/internal/domain/auth"
	"lms-backend/pkg/apperrors"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

const (
	defaultSettingsLanguage = "en"
	defaultSettingsTimezone = "UTC"
)

// GetUserSettings returns persisted account preferences, creating defaults when needed.
func (s *authService) GetUserSettings(ctx context.Context, userID uuid.UUID) (*UserSettingsResult, error) {
	settings, err := s.loadOrCreateUserSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toUserSettingsResult(settings), nil
}

// UpdateUserSettings applies a partial account preference update.
func (s *authService) UpdateUserSettings(ctx context.Context, userID uuid.UUID, cmd UpdateUserSettingsCommand) (*UserSettingsResult, error) {
	settings, err := s.loadOrCreateUserSettings(ctx, userID)
	if err != nil {
		return nil, err
	}

	if cmd.EmailNotifications != nil {
		settings.EmailNotifications = *cmd.EmailNotifications
	}
	if cmd.PushNotifications != nil {
		settings.PushNotifications = *cmd.PushNotifications
	}
	if cmd.NewsletterOptIn != nil {
		settings.NewsletterOptIn = *cmd.NewsletterOptIn
	}
	if cmd.Language != nil {
		language := strings.TrimSpace(*cmd.Language)
		if len(language) < 2 || len(language) > 20 {
			return nil, apperrors.NewSimpleValidationError("INVALID_LANGUAGE", "language must be between 2 and 20 characters")
		}
		settings.Language = language
	}
	if cmd.Timezone != nil {
		timezone := strings.TrimSpace(*cmd.Timezone)
		if !isValidTimezoneName(timezone) {
			return nil, apperrors.NewSimpleValidationError("INVALID_TIMEZONE", "timezone must be a valid IANA timezone name")
		}
		settings.Timezone = timezone
	}

	if err := s.deps.UserSettingsRepo.Upsert(ctx, settings); err != nil {
		return nil, err
	}

	return toUserSettingsResult(settings), nil
}

func (s *authService) loadOrCreateUserSettings(ctx context.Context, userID uuid.UUID) (*domainauth.UserSettings, error) {
	if s.deps.UserSettingsRepo == nil {
		return defaultUserSettings(userID), nil
	}

	settings, err := s.deps.UserSettingsRepo.GetByUserID(ctx, userID)
	if err == nil {
		return settings, nil
	}

	settings = defaultUserSettings(userID)
	if err := s.deps.UserSettingsRepo.Upsert(ctx, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func defaultUserSettings(userID uuid.UUID) *domainauth.UserSettings {
	now := time.Now().UTC()
	return &domainauth.UserSettings{
		UserID:             userID,
		EmailNotifications: true,
		PushNotifications:  true,
		NewsletterOptIn:    false,
		Language:           defaultSettingsLanguage,
		Timezone:           defaultSettingsTimezone,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func toUserSettingsResult(settings *domainauth.UserSettings) *UserSettingsResult {
	return &UserSettingsResult{
		EmailNotifications: settings.EmailNotifications,
		PushNotifications:  settings.PushNotifications,
		NewsletterOptIn:    settings.NewsletterOptIn,
		Language:           settings.Language,
		Timezone:           settings.Timezone,
	}
}

func isValidTimezoneName(value string) bool {
	if value == "" || len(value) > 80 {
		return false
	}
	if value == "UTC" {
		return true
	}
	if strings.Contains(value, " ") || !strings.Contains(value, "/") {
		return false
	}
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		switch r {
		case '/', '_', '-', '+':
			continue
		default:
			return false
		}
	}
	return true
}
