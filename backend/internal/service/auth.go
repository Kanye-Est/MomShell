package service

import (
	"errors"
	"fmt"
	"log"

	"github.com/momshell/backend/internal/config"
	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	pkgjwt "github.com/momshell/backend/pkg/jwt"
	"github.com/momshell/backend/pkg/password"
	"gorm.io/gorm"
)

const (
	errPasswordHashFailed  = "密码加密失败: %w"
	errInvalidRefreshToken = "无效的刷新令牌"
)

type AuthService struct {
	cfg      *config.Config
	userRepo *repository.UserRepo
}

func NewAuthService(cfg *config.Config, userRepo *repository.UserRepo) *AuthService {
	return &AuthService{cfg: cfg, userRepo: userRepo}
}

func (s *AuthService) Register(req dto.RegisterRequest) (*dto.UserResponse, error) {
	// Check if username/email already exists
	exists, err := s.userRepo.ExistsByUsernameOrEmail(req.Username, req.Email)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if exists {
		return nil, errors.New("用户名或邮箱已存在")
	}

	// Hash password
	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf(errPasswordHashFailed, err)
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Nickname:     req.Nickname,
		Role:         model.UserRole(req.Role),
		IsGuest:      false,
		IsActive:     true,
		IsBanned:     false,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return s.buildUserResponse(user), nil
}

func (s *AuthService) Login(req dto.LoginRequest, clientIP string) (*dto.TokenResponse, error) {
	user, err := s.userRepo.FindByUsernameOrEmail(req.Login)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[SECURITY] login_failed | ip=%s | user=%s | detail=user not found", clientIP, req.Login)
			return nil, errors.New("用户名或密码错误")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	if !password.Verify(req.Password, user.PasswordHash) {
		log.Printf("[SECURITY] login_failed | ip=%s | user=%s | detail=wrong password", clientIP, req.Login)
		return nil, errors.New("用户名或密码错误")
	}

	if !user.IsActive {
		log.Printf("[SECURITY] login_failed | ip=%s | user=%s | detail=account disabled", clientIP, req.Login)
		return nil, errors.New("账号已禁用")
	}

	if user.IsBanned {
		log.Printf("[SECURITY] login_failed | ip=%s | user=%s | detail=account banned", clientIP, req.Login)
		return nil, errors.New("账号已被封禁")
	}

	if user.IsAdmin {
		log.Printf("[SECURITY] admin_login | ip=%s | user=%s | detail=admin login successful", clientIP, req.Login)
	}

	return s.generateTokens(user.ID)
}

func (s *AuthService) RefreshToken(refreshToken string) (*dto.TokenResponse, error) {
	claims, err := pkgjwt.ParseToken(refreshToken, s.cfg.JWTSecretKey)
	if err != nil {
		return nil, errors.New(errInvalidRefreshToken)
	}
	if claims.Type != "refresh" {
		return nil, errors.New(errInvalidRefreshToken)
	}

	user, err := s.userRepo.FindByID(claims.Subject)
	if err != nil || !user.IsActive || user.IsBanned {
		return nil, errors.New(errInvalidRefreshToken)
	}

	return s.generateTokens(user.ID)
}

func (s *AuthService) GetCurrentUser(userID string) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	return s.buildUserResponse(user), nil
}

func (s *AuthService) ChangePassword(userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("用户不存在")
	}

	if !password.Verify(oldPassword, user.PasswordHash) {
		return errors.New("原密码错误")
	}

	hash, err := password.Hash(newPassword)
	if err != nil {
		return fmt.Errorf(errPasswordHashFailed, err)
	}

	return s.userRepo.UpdatePassword(userID, hash)
}

func (s *AuthService) ForgotPassword(email string) (string, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		// Don't reveal whether user exists
		return "", nil
	}

	token, err := pkgjwt.CreatePasswordResetToken(user.ID, s.cfg.JWTSecretKey)
	if err != nil {
		return "", fmt.Errorf("创建重置令牌失败: %w", err)
	}

	// In production, send email with reset link
	return token, nil
}

func (s *AuthService) ResetPassword(token, newPassword string) error {
	userID, err := pkgjwt.VerifyPasswordResetToken(token, s.cfg.JWTSecretKey)
	if err != nil {
		return errors.New("无效或过期的重置令牌")
	}

	hash, err := password.Hash(newPassword)
	if err != nil {
		return fmt.Errorf(errPasswordHashFailed, err)
	}

	return s.userRepo.UpdatePassword(userID, hash)
}

func (s *AuthService) UpdateRole(userID, role string) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	// Check if user is a professional
	if model.ProfessionalRoles[user.Role] {
		return nil, errors.New("认证专业人员不能修改角色")
	}
	if user.PartnerID != nil {
		return nil, errors.New("已绑定伴侣，无法更改身份")
	}

	newRole := model.UserRole(role)
	if !model.FamilyRoles[newRole] {
		return nil, errors.New("角色只能是: mom, dad")
	}

	user.Role = newRole
	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("更新角色失败: %w", err)
	}

	return s.buildUserResponse(user), nil
}

func (s *AuthService) CompleteTutorial(userID string) error {
	_, err := s.userRepo.FindByID(userID)
	if err != nil {
		return errors.New("用户不存在")
	}
	return s.userRepo.UpdateTutorialCompleted(userID, true)
}

func (s *AuthService) GetUserByID(userID string) (*model.User, error) {
	return s.userRepo.FindByID(userID)
}

func (s *AuthService) GetJWTSecret() string {
	return s.cfg.JWTSecretKey
}

func (s *AuthService) generateTokens(userID string) (*dto.TokenResponse, error) {
	settings := s.cfg.Runtime()
	accessToken, err := pkgjwt.CreateAccessToken(userID, s.cfg.JWTSecretKey, settings.JWTAccessTokenExpireMin)
	if err != nil {
		return nil, fmt.Errorf("创建访问令牌失败: %w", err)
	}

	refreshToken, err := pkgjwt.CreateRefreshToken(userID, s.cfg.JWTSecretKey, settings.JWTRefreshTokenExpireDays)
	if err != nil {
		return nil, fmt.Errorf("创建刷新令牌失败: %w", err)
	}

	return &dto.TokenResponse{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresIn:        settings.JWTAccessTokenExpireMin * 60,
		RefreshExpiresIn: settings.JWTRefreshTokenExpireDays * 86400,
	}, nil
}

func (s *AuthService) buildUserResponse(user *model.User) *dto.UserResponse {
	resp := &dto.UserResponse{
		ID:                user.ID,
		Username:          user.Username,
		Email:             user.Email,
		Nickname:          user.Nickname,
		AvatarURL:         user.AvatarURL,
		Role:              string(user.Role),
		IsAdmin:           user.IsAdmin,
		TutorialCompleted: user.TutorialCompleted,
		BabyBirthDate:     user.BabyBirthDate,
		PostpartumWeeks:   user.PostpartumWeeks,
		CreatedAt:         user.CreatedAt,
	}

	if user.Certification != nil && user.Certification.Status == model.CertApproved {
		resp.IsCertified = true
		title := ""
		if user.Certification.HospitalOrInstitution != "" {
			title += user.Certification.HospitalOrInstitution
		}
		if user.Certification.Department != nil && *user.Certification.Department != "" {
			title += " " + *user.Certification.Department
		}
		if user.Certification.Title != nil && *user.Certification.Title != "" {
			title += " " + *user.Certification.Title
		}
		if title != "" {
			resp.CertificationTitle = &title
		}
	}

	return resp
}
