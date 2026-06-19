package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"lms-backend/internal/domain/auth"
	"lms-backend/internal/domain/users"
	"lms-backend/pkg/apperrors"
	"lms-backend/pkg/logger"
	"lms-backend/pkg/validator"
	"math/big"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Service defines the users service interface
type Service interface {
	// Student Onboarding
	SubmitStudentProfile(ctx context.Context, userID uuid.UUID, cmd SubmitStudentProfileCommand) (*StudentProfileResult, error)
	GetStudentProfile(ctx context.Context, userID uuid.UUID) (*StudentProfileResult, error)

	// Admin User Management
	ListUsers(ctx context.Context, filters ListUsersFilters) (*ListUsersResult, error)
	CreateUser(ctx context.Context, cmd CreateUserCommand) (*UserResult, error)
	GetUser(ctx context.Context, userID uuid.UUID) (*UserDetailResult, error)
	UpdateUser(ctx context.Context, actorID, targetUserID uuid.UUID, cmd UpdateUserCommand) (*UserResult, error)
	UpdateStudentProfile(ctx context.Context, actorID, targetUserID uuid.UUID, cmd UpdateStudentProfileCommand) error
	ForcePasswordReset(ctx context.Context, actorID, targetUserID uuid.UUID) error
}

// Dependencies for the users service
type ServiceDeps struct {
	UserRepo          auth.UserRepository
	ProfileRepo       users.StudentProfileRepository
	TokenStore        TokenStore
	EmailService      EmailService
	NotificationQueue NotificationQueue
	AuditLogger       AuditLogger
}

// TokenStore interface for token operations
type TokenStore interface {
	DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error
}

// EmailService interface for sending emails
type EmailService interface {
	SendWelcomeEmail(ctx context.Context, email, fullName, tempPassword string) error
	SendPasswordResetEmail(ctx context.Context, email, fullName, resetToken string) error
}

// NotificationQueue interface for enqueueing notifications
type NotificationQueue interface {
	EnqueueNotification(ctx context.Context, userID uuid.UUID, notificationType, title, body string) error
}

// AuditLogger interface for audit logging
type AuditLogger interface {
	LogAction(ctx context.Context, actorID uuid.UUID, actorName, action, targetType string, targetID uuid.UUID, metadata map[string]interface{}, ipAddress string) error
}

// serviceImpl implements the Service interface
type serviceImpl struct {
	deps ServiceDeps
}

type listCapableUserRepo interface {
	List(ctx context.Context, role, status, search *string, fromDate, toDate *time.Time, page, limit int) ([]*auth.User, int, error)
}

// NewService creates a new users service
func NewService(deps ServiceDeps) Service {
	return &serviceImpl{deps: deps}
}

// SubmitStudentProfile validates and persists student profile, setting profile_complete atomically
func (s *serviceImpl) SubmitStudentProfile(ctx context.Context, userID uuid.UUID, cmd SubmitStudentProfileCommand) (*StudentProfileResult, error) {
	// Validate input
	if err := s.validateStudentProfile(cmd); err != nil {
		return nil, err
	}

	// Check if user exists and is a student
	user, err := s.deps.UserRepo.FindByID(ctx, userID)
	if err != nil {
		logger.Error(ctx, "Failed to find user", "error", err, "user_id", userID)
		return nil, apperrors.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}

	if user.Role != "student" {
		return nil, apperrors.NewForbiddenError("NOT_STUDENT", "Only students can submit student profiles")
	}

	// Create profile
	profile := &users.StudentProfile{
		UserID:          userID,
		SchoolName:      cmd.SchoolName,
		ClassGrade:      cmd.ClassGrade,
		RollNumber:      cmd.RollNumber,
		DateOfBirth:     cmd.DateOfBirth,
		Gender:          cmd.Gender,
		GuardianName:    cmd.GuardianName,
		GuardianContact: cmd.GuardianContact,
	}

	// Upsert profile and set profile_complete atomically
	if err := s.deps.ProfileRepo.Upsert(ctx, profile); err != nil {
		logger.Error(ctx, "Failed to upsert student profile", "error", err, "user_id", userID)
		return nil, apperrors.NewInternalError("PROFILE_SAVE_FAILED", "Failed to save student profile")
	}

	logger.Info(ctx, "Student profile submitted successfully", "user_id", userID)

	return &StudentProfileResult{
		UserID:          profile.UserID,
		SchoolName:      profile.SchoolName,
		ClassGrade:      profile.ClassGrade,
		RollNumber:      profile.RollNumber,
		DateOfBirth:     profile.DateOfBirth,
		Gender:          profile.Gender,
		GuardianName:    profile.GuardianName,
		GuardianContact: profile.GuardianContact,
		ProfileComplete: true, // Always true after successful submission
		UpdatedAt:       profile.UpdatedAt,
	}, nil
}

// GetStudentProfile retrieves student profile with profile_complete status
func (s *serviceImpl) GetStudentProfile(ctx context.Context, userID uuid.UUID) (*StudentProfileResult, error) {
	// Check if user exists and is a student
	user, err := s.deps.UserRepo.FindByID(ctx, userID)
	if err != nil {
		logger.Error(ctx, "Failed to find user", "error", err, "user_id", userID)
		return nil, apperrors.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}

	if user.Role != "student" {
		return nil, apperrors.NewForbiddenError("NOT_STUDENT", "Only students have student profiles")
	}

	// Get profile
	profile, err := s.deps.ProfileRepo.FindByUserID(ctx, userID)
	if err != nil {
		// Profile not found - return empty result with profile_complete from user
		return &StudentProfileResult{
			UserID:          userID,
			ProfileComplete: user.ProfileComplete,
		}, nil
	}

	return &StudentProfileResult{
		UserID:          profile.UserID,
		SchoolName:      profile.SchoolName,
		ClassGrade:      profile.ClassGrade,
		RollNumber:      profile.RollNumber,
		DateOfBirth:     profile.DateOfBirth,
		Gender:          profile.Gender,
		GuardianName:    profile.GuardianName,
		GuardianContact: profile.GuardianContact,
		ProfileComplete: user.ProfileComplete,
		UpdatedAt:       profile.UpdatedAt,
	}, nil
}

// validateStudentProfile validates student profile input
func (s *serviceImpl) validateStudentProfile(cmd SubmitStudentProfileCommand) error {
	// Validate school_name (2-200 chars)
	if len(cmd.SchoolName) < 2 || len(cmd.SchoolName) > 200 {
		return apperrors.NewValidationErrorWithDetails("INVALID_SCHOOL_NAME", "School name must be between 2 and 200 characters", nil)
	}

	// Validate class_grade (1-50 chars)
	if len(cmd.ClassGrade) < 1 || len(cmd.ClassGrade) > 50 {
		return apperrors.NewValidationErrorWithDetails("INVALID_CLASS_GRADE", "Class grade must be between 1 and 50 characters", nil)
	}

	// Validate roll_number (1-30 chars)
	if len(cmd.RollNumber) < 1 || len(cmd.RollNumber) > 30 {
		return apperrors.NewValidationErrorWithDetails("INVALID_ROLL_NUMBER", "Roll number must be between 1 and 30 characters", nil)
	}

	// Validate date_of_birth (must be in the past)
	if cmd.DateOfBirth.After(time.Now()) {
		return apperrors.NewValidationErrorWithDetails("INVALID_DATE_OF_BIRTH", "Date of birth must be in the past", nil)
	}

	// Validate age (must be <= 30 years)
	age := time.Now().Year() - cmd.DateOfBirth.Year()
	if age > 30 {
		return apperrors.NewValidationErrorWithDetails("INVALID_AGE", "Age must be 30 years or less", nil)
	}

	// Validate gender if provided
	if cmd.Gender != nil {
		validGenders := []string{"male", "female", "other", "prefer_not_to_say"}
		if !validator.IsInEnum(*cmd.Gender, validGenders) {
			return apperrors.NewValidationErrorWithDetails("INVALID_GENDER", "Gender must be one of: male, female, other, prefer_not_to_say", nil)
		}
	}

	return nil
}

// ListUsers returns paginated list of users with filters
func (s *serviceImpl) ListUsers(ctx context.Context, filters ListUsersFilters) (*ListUsersResult, error) {
	repo, ok := s.deps.UserRepo.(listCapableUserRepo)
	if !ok {
		return &ListUsersResult{
			Users: []UserResult{},
			Meta: PaginationMetadata{
				Page:       filters.Page,
				Limit:      filters.Limit,
				Total:      0,
				TotalPages: 0,
			},
		}, nil
	}

	if filters.Page < 1 {
		filters.Page = 1
	}
	if filters.Limit < 1 || filters.Limit > 100 {
		filters.Limit = 20
	}

	userRows, total, err := repo.List(ctx, filters.Role, filters.Status, filters.Search, filters.FromDate, filters.ToDate, filters.Page, filters.Limit)
	if err != nil {
		logger.Error(ctx, "Failed to list users", "error", err)
		return nil, apperrors.NewInternalError("USER_LIST_FAILED", "Failed to list users")
	}

	results := make([]UserResult, 0, len(userRows))
	for _, user := range userRows {
		results = append(results, UserResult{
			ID:              user.ID,
			FullName:        user.FullName,
			Email:           user.Email,
			Role:            user.Role,
			Status:          user.Status,
			ProfileComplete: user.ProfileComplete,
			LastSignInAt:    user.LastSignInAt,
			CreatedAt:       user.CreatedAt,
			UpdatedAt:       user.UpdatedAt,
		})
	}

	totalPages := (total + filters.Limit - 1) / filters.Limit
	return &ListUsersResult{
		Users: results,
		Meta: PaginationMetadata{
			Page:       filters.Page,
			Limit:      filters.Limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// CreateUser creates a new active user account and sends welcome email
func (s *serviceImpl) CreateUser(ctx context.Context, cmd CreateUserCommand) (*UserResult, error) {
	// Validate input
	if err := s.validateCreateUser(cmd); err != nil {
		return nil, err
	}

	// Check if email already exists
	existingUser, _ := s.deps.UserRepo.FindByEmail(ctx, cmd.Email)
	if existingUser != nil {
		return nil, apperrors.NewConflictError("EMAIL_EXISTS", "Email already exists")
	}

	// Generate temporary password
	tempPassword, err := generateTempPassword()
	if err != nil {
		logger.Error(ctx, "Failed to generate temp password", "error", err)
		return nil, apperrors.NewInternalError("PASSWORD_GENERATION_FAILED", "Failed to generate temporary password")
	}

	// Hash password
	passwordHash, err := hashPassword(tempPassword)
	if err != nil {
		logger.Error(ctx, "Failed to hash password", "error", err)
		return nil, apperrors.NewInternalError("PASSWORD_HASH_FAILED", "Failed to hash password")
	}

	// Create user
	user := &auth.User{
		FullName:        cmd.FullName,
		Email:           cmd.Email,
		PasswordHash:    &passwordHash,
		Role:            cmd.Role,
		Status:          "active",
		ProfileComplete: false,
	}

	if err := s.deps.UserRepo.Create(ctx, user); err != nil {
		logger.Error(ctx, "Failed to create user", "error", err)
		return nil, apperrors.NewInternalError("USER_CREATION_FAILED", "Failed to create user")
	}

	// Send welcome email with temp credentials
	if err := s.deps.EmailService.SendWelcomeEmail(ctx, user.Email, user.FullName, tempPassword); err != nil {
		logger.Error(ctx, "Failed to send welcome email", "error", err, "user_id", user.ID)
		// Don't fail the operation if email fails
	}

	logger.Info(ctx, "User created successfully", "user_id", user.ID, "role", user.Role)

	return &UserResult{
		ID:              user.ID,
		FullName:        user.FullName,
		Email:           user.Email,
		Role:            user.Role,
		Status:          user.Status,
		ProfileComplete: user.ProfileComplete,
		LastSignInAt:    user.LastSignInAt,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}, nil
}

// GetUser retrieves full user profile including student_profile for students
func (s *serviceImpl) GetUser(ctx context.Context, userID uuid.UUID) (*UserDetailResult, error) {
	// Get user
	user, err := s.deps.UserRepo.FindByID(ctx, userID)
	if err != nil {
		logger.Error(ctx, "Failed to find user", "error", err, "user_id", userID)
		return nil, apperrors.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}

	result := &UserDetailResult{
		ID:              user.ID,
		FullName:        user.FullName,
		Email:           user.Email,
		Role:            user.Role,
		Status:          user.Status,
		ProfileComplete: user.ProfileComplete,
		LastSignInAt:    user.LastSignInAt,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}

	// If student, include student profile
	if user.Role == "student" {
		profile, err := s.deps.ProfileRepo.FindByUserID(ctx, userID)
		if err == nil {
			result.StudentProfile = &StudentProfileResult{
				UserID:          profile.UserID,
				SchoolName:      profile.SchoolName,
				ClassGrade:      profile.ClassGrade,
				RollNumber:      profile.RollNumber,
				DateOfBirth:     profile.DateOfBirth,
				Gender:          profile.Gender,
				GuardianName:    profile.GuardianName,
				GuardianContact: profile.GuardianContact,
				ProfileComplete: user.ProfileComplete,
				UpdatedAt:       profile.UpdatedAt,
			}
		}
	}

	return result, nil
}

// UpdateUser updates user role or status with audit logging
func (s *serviceImpl) UpdateUser(ctx context.Context, actorID, targetUserID uuid.UUID, cmd UpdateUserCommand) (*UserResult, error) {
	// Prevent self-role-change
	if actorID == targetUserID && cmd.Role != nil {
		return nil, apperrors.NewForbiddenError("SELF_ROLE_CHANGE_FORBIDDEN", "Cannot change your own role")
	}

	// Get target user
	user, err := s.deps.UserRepo.FindByID(ctx, targetUserID)
	if err != nil {
		logger.Error(ctx, "Failed to find user", "error", err, "user_id", targetUserID)
		return nil, apperrors.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}

	// Get actor for audit logging
	actor, err := s.deps.UserRepo.FindByID(ctx, actorID)
	if err != nil {
		logger.Error(ctx, "Failed to find actor", "error", err, "actor_id", actorID)
		return nil, apperrors.NewInternalError("ACTOR_NOT_FOUND", "Actor not found")
	}

	// Track changes for audit log
	metadata := make(map[string]interface{})
	oldRole := user.Role
	oldStatus := user.Status

	// Update role if provided
	if cmd.Role != nil {
		if err := s.validateRole(*cmd.Role); err != nil {
			return nil, err
		}
		user.Role = *cmd.Role
		metadata["from_role"] = oldRole
		metadata["to_role"] = *cmd.Role
	}

	// Update status if provided
	if cmd.Status != nil {
		if err := s.validateStatus(*cmd.Status); err != nil {
			return nil, err
		}
		user.Status = *cmd.Status
		metadata["from_status"] = oldStatus
		metadata["to_status"] = *cmd.Status
	}

	// Update user
	if err := s.deps.UserRepo.Update(ctx, user); err != nil {
		logger.Error(ctx, "Failed to update user", "error", err, "user_id", targetUserID)
		return nil, apperrors.NewInternalError("USER_UPDATE_FAILED", "Failed to update user")
	}

	// Record audit log for role change
	if cmd.Role != nil && oldRole != *cmd.Role {
		if err := s.deps.AuditLogger.LogAction(ctx, actorID, actor.FullName, "role_changed", "user", targetUserID, metadata, ""); err != nil {
			logger.Error(ctx, "Failed to log audit action", "error", err)
		}

		// Send notification to user
		notificationBody := fmt.Sprintf("Your role has been changed from %s to %s", oldRole, *cmd.Role)
		if err := s.deps.NotificationQueue.EnqueueNotification(ctx, targetUserID, "role_changed", "Role Changed", notificationBody); err != nil {
			logger.Error(ctx, "Failed to enqueue notification", "error", err)
		}
	}

	// Invalidate all refresh tokens on deactivation
	if cmd.Status != nil && *cmd.Status == "inactive" && oldStatus != "inactive" {
		if err := s.deps.TokenStore.DeleteAllRefreshTokens(ctx, targetUserID); err != nil {
			logger.Error(ctx, "Failed to delete refresh tokens", "error", err, "user_id", targetUserID)
		}

		// Record audit log for deactivation
		if err := s.deps.AuditLogger.LogAction(ctx, actorID, actor.FullName, "user_deactivated", "user", targetUserID, metadata, ""); err != nil {
			logger.Error(ctx, "Failed to log audit action", "error", err)
		}
	}

	logger.Info(ctx, "User updated successfully", "user_id", targetUserID, "actor_id", actorID)

	return &UserResult{
		ID:              user.ID,
		FullName:        user.FullName,
		Email:           user.Email,
		Role:            user.Role,
		Status:          user.Status,
		ProfileComplete: user.ProfileComplete,
		LastSignInAt:    user.LastSignInAt,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
	}, nil
}

// UpdateStudentProfile updates student profile by admin with audit logging
func (s *serviceImpl) UpdateStudentProfile(ctx context.Context, actorID, targetUserID uuid.UUID, cmd UpdateStudentProfileCommand) error {
	// Get target user
	user, err := s.deps.UserRepo.FindByID(ctx, targetUserID)
	if err != nil {
		logger.Error(ctx, "Failed to find user", "error", err, "user_id", targetUserID)
		return apperrors.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}

	if user.Role != "student" {
		return apperrors.NewForbiddenError("NOT_STUDENT", "Target user is not a student")
	}

	// Get existing profile
	profile, err := s.deps.ProfileRepo.FindByUserID(ctx, targetUserID)
	if err != nil {
		logger.Error(ctx, "Failed to find student profile", "error", err, "user_id", targetUserID)
		return apperrors.NewNotFoundError("PROFILE_NOT_FOUND", "Student profile not found")
	}

	// Update fields if provided
	if cmd.SchoolName != nil {
		profile.SchoolName = *cmd.SchoolName
	}
	if cmd.ClassGrade != nil {
		profile.ClassGrade = *cmd.ClassGrade
	}
	if cmd.RollNumber != nil {
		profile.RollNumber = *cmd.RollNumber
	}
	if cmd.DateOfBirth != nil {
		profile.DateOfBirth = *cmd.DateOfBirth
	}
	if cmd.Gender != nil {
		profile.Gender = cmd.Gender
	}
	if cmd.GuardianName != nil {
		profile.GuardianName = cmd.GuardianName
	}
	if cmd.GuardianContact != nil {
		profile.GuardianContact = cmd.GuardianContact
	}

	// Update profile
	if err := s.deps.ProfileRepo.UpdateByAdmin(ctx, profile); err != nil {
		logger.Error(ctx, "Failed to update student profile", "error", err, "user_id", targetUserID)
		return apperrors.NewInternalError("PROFILE_UPDATE_FAILED", "Failed to update student profile")
	}

	// Get actor for audit logging
	actor, err := s.deps.UserRepo.FindByID(ctx, actorID)
	if err != nil {
		logger.Error(ctx, "Failed to find actor", "error", err, "actor_id", actorID)
		return apperrors.NewInternalError("ACTOR_NOT_FOUND", "Actor not found")
	}

	// Record audit log
	metadata := map[string]interface{}{
		"updated_fields": cmd,
	}
	if err := s.deps.AuditLogger.LogAction(ctx, actorID, actor.FullName, "student_profile_edited", "student_profile", targetUserID, metadata, ""); err != nil {
		logger.Error(ctx, "Failed to log audit action", "error", err)
	}

	// Send in-app notification
	notificationBody := "Your student profile has been updated by an administrator"
	if err := s.deps.NotificationQueue.EnqueueNotification(ctx, targetUserID, "profile_updated", "Profile Updated", notificationBody); err != nil {
		logger.Error(ctx, "Failed to enqueue notification", "error", err)
	}

	logger.Info(ctx, "Student profile updated by admin", "user_id", targetUserID, "actor_id", actorID)

	return nil
}

// ForcePasswordReset generates reset token and sends email
func (s *serviceImpl) ForcePasswordReset(ctx context.Context, actorID, targetUserID uuid.UUID) error {
	// Get target user
	user, err := s.deps.UserRepo.FindByID(ctx, targetUserID)
	if err != nil {
		logger.Error(ctx, "Failed to find user", "error", err, "user_id", targetUserID)
		return apperrors.NewNotFoundError("USER_NOT_FOUND", "User not found")
	}

	// Generate reset token
	resetToken, err := generateResetToken()
	if err != nil {
		logger.Error(ctx, "Failed to generate reset token", "error", err)
		return apperrors.NewInternalError("TOKEN_GENERATION_FAILED", "Failed to generate reset token")
	}

	// Send password reset email
	if err := s.deps.EmailService.SendPasswordResetEmail(ctx, user.Email, user.FullName, resetToken); err != nil {
		logger.Error(ctx, "Failed to send password reset email", "error", err, "user_id", targetUserID)
		return apperrors.NewInternalError("EMAIL_SEND_FAILED", "Failed to send password reset email")
	}

	logger.Info(ctx, "Password reset forced", "user_id", targetUserID, "actor_id", actorID)

	return nil
}

// Helper functions

func (s *serviceImpl) validateCreateUser(cmd CreateUserCommand) error {
	if len(cmd.FullName) < 2 || len(cmd.FullName) > 100 {
		return apperrors.NewValidationErrorWithDetails("INVALID_FULL_NAME", "Full name must be between 2 and 100 characters", nil)
	}

	if !validator.IsValidEmail(cmd.Email) {
		return apperrors.NewValidationErrorWithDetails("INVALID_EMAIL", "Invalid email address", nil)
	}

	return s.validateRole(cmd.Role)
}

func (s *serviceImpl) validateRole(role string) error {
	validRoles := []string{"student", "teacher", "admin"}
	if !validator.IsInEnum(role, validRoles) {
		return apperrors.NewValidationErrorWithDetails("INVALID_ROLE", "Role must be one of: student, teacher, admin", nil)
	}
	return nil
}

func (s *serviceImpl) validateStatus(status string) error {
	validStatuses := []string{"active", "inactive"}
	if !validator.IsInEnum(status, validStatuses) {
		return apperrors.NewValidationErrorWithDetails("INVALID_STATUS", "Status must be one of: active, inactive", nil)
	}
	return nil
}

func generateTempPassword() (string, error) {
	// Generate a random 12-character password
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	password := make([]byte, 12)
	for i := range password {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		password[i] = charset[num.Int64()]
	}
	return string(password), nil
}

func generateResetToken() (string, error) {
	// Generate a random 32-byte token
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return hex.EncodeToString(token), nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
