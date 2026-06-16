package auth

import "github.com/google/uuid"

// RegisterCommand represents a registration request
type RegisterCommand struct {
	FullName        string `json:"full_name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
	Role            string `json:"role"` // student, teacher
}

// VerifyOTPCommand represents an OTP verification request
type VerifyOTPCommand struct {
	Email   string `json:"email"`
	OTP     string `json:"otp"`
	Purpose string `json:"purpose"`
}

// ResendOTPCommand represents an OTP resend request
type ResendOTPCommand struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"`
}

// LoginCommand represents a login request
type LoginCommand struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

// AdminLoginCommand represents an admin login request
type AdminLoginCommand struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AdminVerifyOTPCommand represents an admin OTP verification request
type AdminVerifyOTPCommand struct {
	OTPSessionToken string `json:"otp_session_token"`
	OTP             string `json:"otp"`
}

// ImpersonateUserCommand represents an admin request to view the platform as another user.
type ImpersonateUserCommand struct {
	ActorID      uuid.UUID `json:"actor_id"`
	TargetUserID uuid.UUID `json:"target_user_id"`
}

// ForgotPasswordCommand represents a forgot password request
type ForgotPasswordCommand struct {
	Email string `json:"email"`
}

// ResetPasswordCommand represents a password reset request
type ResetPasswordCommand struct {
	Token           string `json:"token"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

// ChangePasswordCommand represents a password change request
type ChangePasswordCommand struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

// UpdateProfileCommand represents a profile update request
type UpdateProfileCommand struct {
	FullName *string `json:"full_name,omitempty"`
}

// UpdateUserSettingsCommand represents a partial account preference update.
type UpdateUserSettingsCommand struct {
	EmailNotifications *bool   `json:"email_notifications,omitempty"`
	PushNotifications  *bool   `json:"push_notifications,omitempty"`
	NewsletterOptIn    *bool   `json:"newsletter_opt_in,omitempty"`
	Language           *string `json:"language,omitempty"`
	Timezone           *string `json:"timezone,omitempty"`
}

// OAuthRedirectCommand represents an OAuth redirect request
type OAuthRedirectCommand struct {
	Provider string `json:"provider"`
	ReturnTo string `json:"return_to,omitempty"`
}

// OAuthCallbackCommand represents an OAuth callback request
type OAuthCallbackCommand struct {
	Provider string `json:"provider"`
	Code     string `json:"code"`
	State    string `json:"state"`
}

// ConnectProviderCommand represents a provider connection request
type ConnectProviderCommand struct {
	Provider string `json:"provider"`
	Code     string `json:"code"`
	State    string `json:"state"`
}

// DisconnectProviderCommand represents a provider disconnection request
type DisconnectProviderCommand struct {
	Provider string `json:"provider"`
}
