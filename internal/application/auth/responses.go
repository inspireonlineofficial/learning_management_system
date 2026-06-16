package auth

import "time"

// TokenPair represents an access token and refresh token pair
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

// ImpersonationResult returns a session scoped to the impersonated user.
type ImpersonationResult struct {
	TokenPair
	User ProfileResult `json:"user"`
}

// RegisterResult represents the result of a registration
type RegisterResult struct {
	Message   string    `json:"message"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

// OTPSessionResult represents the result of admin login (before OTP verification)
type OTPSessionResult struct {
	OTPSessionToken string    `json:"otp_session_token"`
	MaskedEmail     string    `json:"masked_email"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// OAuthResult represents the result of an OAuth callback
type OAuthResult struct {
	TokenPair
	IsNewUser bool   `json:"is_new_user"`
	ReturnTo  string `json:"return_to,omitempty"`
}

// ProfileResult represents a user profile
type ProfileResult struct {
	ID              string    `json:"id"`
	FullName        string    `json:"full_name"`
	Email           string    `json:"email"`
	Role            string    `json:"role"`
	ProfileComplete bool      `json:"profile_complete"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UserSettingsResult represents account preference settings.
type UserSettingsResult struct {
	EmailNotifications bool   `json:"email_notifications"`
	PushNotifications  bool   `json:"push_notifications"`
	NewsletterOptIn    bool   `json:"newsletter_opt_in"`
	Language           string `json:"language"`
	Timezone           string `json:"timezone"`
}

// OAuthProviderResult represents a linked OAuth provider
type OAuthProviderResult struct {
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
}
