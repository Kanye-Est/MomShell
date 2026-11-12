package dto

import "time"

// RegisterRequest is the request body for registration
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Nickname string `json:"nickname" binding:"required,min=1,max=50"`
	Role     string `json:"role" binding:"omitempty,oneof=mom dad family"`
}

// LoginRequest is the request body for login
type LoginRequest struct {
	Login    string `json:"login" binding:"required"` // username or email
	Password string `json:"password" binding:"required"`
}

// TokenResponse is the response body for login/refresh (internal use)
type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"` // seconds
	RefreshExpiresIn int    `json:"-"`          // seconds
}

// AccessTokenResponse is the response body sent to clients (refresh token in cookie only)
type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // seconds
}

// RefreshRequest is the request body for token refresh
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UserResponse is the response body for user info
type UserResponse struct {
	ID                 string     `json:"id"`
	Username           string     `json:"username"`
	Email              string     `json:"email"`
	Nickname           string     `json:"nickname"`
	AvatarURL          *string    `json:"avatar_url"`
	Role               string     `json:"role"`
	DadChatStyle       string     `json:"dad_chat_style"`
	IsAdmin            bool       `json:"is_admin"`
	IsCertified        bool       `json:"is_certified"`
	TutorialCompleted  bool       `json:"tutorial_completed"`
	CertificationTitle *string    `json:"certification_title"`
	BabyBirthDate      *time.Time `json:"baby_birth_date"`
	PostpartumWeeks    *int       `json:"postpartum_weeks"`
	CreatedAt          time.Time  `json:"created_at"`
}

// ChangePasswordRequest is the request body for changing password
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ForgotPasswordRequest is the request body for forgot password
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest is the request body for resetting password
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// UpdateRoleRequest is the request body for updating user role
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=mom dad"`
}
