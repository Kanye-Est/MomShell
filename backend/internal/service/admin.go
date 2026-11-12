package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/momshell/backend/internal/config"
	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/fileutil"
	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	"github.com/momshell/backend/pkg/password"
)

type AdminService struct {
	cfg       *config.Config
	adminRepo *repository.AdminRepo
	userRepo  *repository.UserRepo
	photoRepo *repository.PhotoRepo
}

func NewAdminService(cfg *config.Config, adminRepo *repository.AdminRepo, userRepo *repository.UserRepo, photoRepo *repository.PhotoRepo) *AdminService {
	return &AdminService{
		cfg:       cfg,
		adminRepo: adminRepo,
		userRepo:  userRepo,
		photoRepo: photoRepo,
	}
}

// GetDashboardStats returns aggregated statistics for the dashboard
func (s *AdminService) GetDashboardStats() (*dto.DashboardStats, error) {
	total, active, banned, guest, err := s.adminRepo.CountUsers()
	if err != nil {
		return nil, fmt.Errorf("统计用户失败: %w", err)
	}

	roleDistribution, err := s.adminRepo.CountUsersByRole()
	if err != nil {
		return nil, fmt.Errorf("统计角色分布失败: %w", err)
	}

	questions, err := s.adminRepo.CountQuestions()
	if err != nil {
		return nil, fmt.Errorf("统计问题失败: %w", err)
	}

	answers, err := s.adminRepo.CountAnswers()
	if err != nil {
		return nil, fmt.Errorf("统计回答失败: %w", err)
	}

	certifications, err := s.adminRepo.CountCertifications()
	if err != nil {
		return nil, fmt.Errorf("统计认证失败: %w", err)
	}

	totalPhotos, wallPhotos, err := s.adminRepo.CountPhotos()
	if err != nil {
		return nil, fmt.Errorf("统计照片失败: %w", err)
	}

	return &dto.DashboardStats{
		TotalUsers:          total,
		ActiveUsers:         active,
		BannedUsers:         banned,
		GuestUsers:          guest,
		RoleDistribution:    roleDistribution,
		TotalQuestions:      questions,
		TotalAnswers:        answers,
		TotalCertifications: certifications,
		TotalPhotos:         totalPhotos,
		WallPhotos:          wallPhotos,
	}, nil
}

// ListUsers returns a paginated list of users
func (s *AdminService) ListUsers(params dto.AdminUserListParams) (*dto.PaginatedResponse, error) {
	offset := params.GetOffset()
	limit := params.GetPageSize()

	users, total, err := s.adminRepo.ListUsers(params.Search, params.Role, params.Status, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}

	items := make([]dto.AdminUserListItem, len(users))
	for i, u := range users {
		items[i] = dto.AdminUserListItem{
			ID:                u.ID,
			Username:          u.Username,
			Email:             u.Email,
			Nickname:          u.Nickname,
			Role:              string(u.Role),
			IsAdmin:           u.IsAdmin,
			IsActive:          u.IsActive,
			IsBanned:          u.IsBanned,
			IsGuest:           u.IsGuest,
			TutorialCompleted: u.TutorialCompleted,
			CreatedAt:         u.CreatedAt,
		}
	}

	resp := dto.NewPaginatedResponse(items, total, params.GetPage(), params.GetPageSize())
	return &resp, nil
}

// GetUser returns a single user's detail
func (s *AdminService) GetUser(id string) (*dto.AdminUserDetail, error) {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	detail := &dto.AdminUserDetail{
		ID:                user.ID,
		Username:          user.Username,
		Email:             user.Email,
		Nickname:          user.Nickname,
		AvatarURL:         user.AvatarURL,
		Role:              string(user.Role),
		IsAdmin:           user.IsAdmin,
		ShellCode:         user.ShellCode,
		IsGuest:           user.IsGuest,
		IsActive:          user.IsActive,
		IsBanned:          user.IsBanned,
		TutorialCompleted: user.TutorialCompleted,
		PartnerID:         user.PartnerID,
		BabyBirthDate:     user.BabyBirthDate,
		CreatedAt:         user.CreatedAt,
		UpdatedAt:         user.UpdatedAt,
		LastActiveAt:      user.LastActiveAt,
	}

	if user.Certification != nil {
		status := string(user.Certification.Status)
		certType := string(user.Certification.CertificationType)
		detail.CertificationStatus = &status
		detail.CertificationType = &certType
	}

	return detail, nil
}

// CreateUser creates a new user from admin panel
func (s *AdminService) CreateUser(req dto.AdminCreateUser) (*dto.AdminUserDetail, error) {
	exists, err := s.userRepo.ExistsByUsernameOrEmail(req.Username, req.Email)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if exists {
		return nil, errors.New("用户名或邮箱已存在")
	}

	hash, err := password.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Nickname:     req.Nickname,
		Role:         model.UserRole(req.Role),
		IsActive:     true,
		IsBanned:     false,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return s.GetUser(user.ID)
}

// validateSelfUpdate checks that an admin is not demoting, deactivating, or banning themselves.
func validateSelfUpdate(req dto.AdminUserUpdate) error {
	if req.IsAdmin != nil && !*req.IsAdmin {
		return errors.New("不能取消自己的管理员权限")
	}
	if req.IsActive != nil && !*req.IsActive {
		return errors.New("不能停用自己的账号")
	}
	if req.IsBanned != nil && *req.IsBanned {
		return errors.New("不能封禁自己的账号")
	}
	return nil
}

// applyUserUpdates applies the non-nil fields from req onto the user model.
func applyUserUpdates(user *model.User, req dto.AdminUserUpdate) {
	if req.Role != nil {
		user.Role = model.UserRole(*req.Role)
	}
	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.IsBanned != nil {
		user.IsBanned = *req.IsBanned
	}
	if req.Nickname != nil {
		user.Nickname = *req.Nickname
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
}

// UpdateUser updates user fields (role, is_active, is_banned, nickname, email)
func (s *AdminService) UpdateUser(id, adminID string, req dto.AdminUserUpdate) (*dto.AdminUserDetail, error) {
	if id == adminID {
		if err := validateSelfUpdate(req); err != nil {
			return nil, err
		}
	}

	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	applyUserUpdates(user, req)

	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	return s.GetUser(user.ID)
}

// DeleteUser hard-deletes a user (prevents self-deletion)
func (s *AdminService) DeleteUser(id, adminID string) error {
	if id == adminID {
		return errors.New("不能删除自己的账号")
	}

	_, err := s.userRepo.FindByID(id)
	if err != nil {
		return errors.New("用户不存在")
	}

	return s.adminRepo.DeleteUser(id)
}

// ListPhotos returns a paginated list of all photos for admin
func (s *AdminService) ListPhotos(params dto.AdminPhotoListParams) (*dto.PaginatedResponse, error) {
	offset := params.GetOffset()
	limit := params.GetPageSize()

	photos, total, err := s.photoRepo.FindAllPaginated(params.Search, params.UserID, params.Source, params.OnWall, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("查询照片列表失败: %w", err)
	}

	items := make([]dto.AdminPhotoListItem, len(photos))
	for i, p := range photos {
		items[i] = dto.AdminPhotoListItem{
			ID:        p.ID,
			UserID:    p.UserID,
			Username:  p.User.Username,
			Title:     p.Title,
			ImageURL:  p.ImageURL,
			IsOnWall:  p.IsOnWall,
			Source:    p.Source,
			CreatedAt: p.CreatedAt,
		}
	}

	resp := dto.NewPaginatedResponse(items, total, params.GetPage(), params.GetPageSize())
	return &resp, nil
}

// DeletePhoto deletes a photo by ID (admin, no user scope) and removes the disk file.
func (s *AdminService) DeletePhoto(id string) error {
	photo, err := s.photoRepo.FindByID(id)
	if err != nil {
		return errors.New("照片不存在")
	}

	if photo.ImageURL != "" {
		fileutil.RemoveUploadedFile(photo.ImageURL)
	}

	return s.photoRepo.DeleteByID(id)
}

// maskString masks a string, showing first n and last m characters
func maskString(s string, showPrefix, showSuffix int) string {
	if len(s) <= showPrefix+showSuffix {
		return strings.Repeat("*", len(s))
	}
	return s[:showPrefix] + strings.Repeat("*", len(s)-showPrefix-showSuffix) + s[len(s)-showSuffix:]
}

// GetConfig returns configuration items with sensitive values masked
func (s *AdminService) GetConfig() []dto.ConfigItem {
	settings := s.cfg.Runtime()

	return []dto.ConfigItem{
		// Read-only
		{Key: "DATABASE_URL", Value: maskString(s.cfg.DatabaseURL, 0, 0), Editable: false},
		{Key: "JWT_ALGORITHM", Value: s.cfg.JWTAlgorithm, Editable: false},
		{Key: "PORT", Value: s.cfg.Port, Editable: false},
		// Editable
		{Key: "OPENAI_API_KEY", Value: maskString(settings.OpenAIAPIKey, 3, 4), Editable: true},
		{Key: "OPENAI_BASE_URL", Value: settings.OpenAIBaseURL, Editable: true},
		{Key: "OPENAI_MODEL", Value: settings.OpenAIModel, Editable: true},
		{Key: "JWT_ACCESS_TOKEN_EXPIRE_MINUTES", Value: strconv.Itoa(settings.JWTAccessTokenExpireMin), Editable: true},
		{Key: "JWT_REFRESH_TOKEN_EXPIRE_DAYS", Value: strconv.Itoa(settings.JWTRefreshTokenExpireDays), Editable: true},
	}
}

// UpdateConfig updates editable configuration items at runtime
func (s *AdminService) UpdateConfig(req dto.ConfigUpdateRequest) error {
	return s.cfg.UpdateRuntime(req.Items)
}
