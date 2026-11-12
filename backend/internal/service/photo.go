package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/fileutil"
	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	"github.com/momshell/backend/pkg/openai"
)

const (
	maxPhotosPerFamily = 50
	maxWallPhotos      = 10
	errPhotoNotFound   = "photo not found"
)

type PhotoService struct {
	photoRepo  *repository.PhotoRepo
	userRepo   *repository.UserRepo
	aiClient   *openai.Client
	imageModel string
}

func NewPhotoService(photoRepo *repository.PhotoRepo, userRepo *repository.UserRepo, aiClient *openai.Client, imageModel string) *PhotoService {
	return &PhotoService{
		photoRepo:  photoRepo,
		userRepo:   userRepo,
		aiClient:   aiClient,
		imageModel: imageModel,
	}
}

func (s *PhotoService) getFamilyIDs(userID string) []string {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return []string{userID}
	}
	if user.PartnerID != nil && *user.PartnerID != "" {
		return []string{userID, *user.PartnerID}
	}
	return []string{userID}
}

func (s *PhotoService) buildNicknameMap(familyIDs []string) map[string]string {
	m := make(map[string]string, len(familyIDs))
	for _, id := range familyIDs {
		if u, err := s.userRepo.FindByID(id); err == nil {
			m[id] = u.Nickname
		}
	}
	return m
}

func (s *PhotoService) ListPhotos(userID string, page, pageSize int) (*dto.PhotoListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	familyIDs := s.getFamilyIDs(userID)
	photos, total, err := s.photoRepo.FindByFamilyIDs(familyIDs, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list photos: %w", err)
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	nicknameMap := s.buildNicknameMap(familyIDs)

	items := make([]dto.PhotoResponse, 0, len(photos))
	for _, p := range photos {
		items = append(items, toPhotoResponse(p, nicknameMap[p.UserID]))
	}

	return &dto.PhotoListResponse{
		Photos:     items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (s *PhotoService) GetPhoto(id, userID string) (*dto.PhotoResponse, error) {
	familyIDs := s.getFamilyIDs(userID)
	photo, err := s.photoRepo.FindByIDAndFamilyIDs(id, familyIDs)
	if err != nil {
		return nil, errors.New(errPhotoNotFound)
	}
	nicknameMap := s.buildNicknameMap(familyIDs)
	resp := toPhotoResponse(*photo, nicknameMap[photo.UserID])
	return &resp, nil
}

func (s *PhotoService) CreateFromUpload(userID, title, imageURL string) (*dto.PhotoResponse, error) {
	familyIDs := s.getFamilyIDs(userID)
	count, err := s.photoRepo.CountByFamilyIDs(familyIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to check photo count: %w", err)
	}
	if count >= maxPhotosPerFamily {
		return nil, fmt.Errorf("photo limit reached (max %d)", maxPhotosPerFamily)
	}

	photo := &model.Photo{
		UserID:   userID,
		Title:    title,
		ImageURL: imageURL,
		Source:   "upload",
	}

	if err := s.photoRepo.Create(photo); err != nil {
		return nil, fmt.Errorf("failed to create photo: %w", err)
	}

	nicknameMap := s.buildNicknameMap(familyIDs)
	resp := toPhotoResponse(*photo, nicknameMap[userID])
	return &resp, nil
}

func (s *PhotoService) GeneratePhoto(ctx context.Context, userID string, req dto.GeneratePhotoRequest) (*dto.PhotoResponse, error) {
	if s.imageModel == "" {
		return nil, openai.ErrImageNotConfigured
	}

	familyIDs := s.getFamilyIDs(userID)
	count, err := s.photoRepo.CountByFamilyIDs(familyIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to check photo count: %w", err)
	}
	if count >= maxPhotosPerFamily {
		return nil, fmt.Errorf("photo limit reached (max %d)", maxPhotosPerFamily)
	}

	var userRole string
	if user, err := s.userRepo.FindByID(userID); err == nil {
		userRole = string(user.Role)
	}

	styledPrompt := buildImagePrompt(req.Prompt, userRole)
	imgResp, err := s.aiClient.GenerateImage(ctx, s.imageModel, styledPrompt)
	if err != nil {
		return nil, fmt.Errorf("image generation failed: %w", err)
	}

	imageURL, err := s.downloadGeneratedImage(imgResp)
	if err != nil {
		return nil, fmt.Errorf("failed to save generated image: %w", err)
	}

	photo := &model.Photo{
		UserID:      userID,
		Title:       truncate(req.Prompt, 200),
		Description: req.Prompt,
		ImageURL:    imageURL,
		Source:      "ai_generated",
	}

	if err := s.photoRepo.Create(photo); err != nil {
		return nil, fmt.Errorf("failed to create photo: %w", err)
	}

	nicknameMap := s.buildNicknameMap(familyIDs)
	resp := toPhotoResponse(*photo, nicknameMap[userID])
	return &resp, nil
}

func (s *PhotoService) UpdatePhoto(id, userID string, req dto.UpdatePhotoRequest) (*dto.PhotoResponse, error) {
	familyIDs := s.getFamilyIDs(userID)
	photo, err := s.photoRepo.FindByIDAndFamilyIDs(id, familyIDs)
	if err != nil {
		return nil, errors.New(errPhotoNotFound)
	}

	if req.Title != nil {
		photo.Title = *req.Title
	}
	if req.Description != nil {
		photo.Description = *req.Description
	}
	if req.Tags != nil {
		tagsJSON, jsonErr := json.Marshal(req.Tags)
		if jsonErr != nil {
			return nil, fmt.Errorf("failed to encode tags: %w", jsonErr)
		}
		photo.Tags = string(tagsJSON)
	}

	if err := s.photoRepo.Update(photo); err != nil {
		return nil, fmt.Errorf("failed to update photo: %w", err)
	}

	nicknameMap := s.buildNicknameMap(familyIDs)
	resp := toPhotoResponse(*photo, nicknameMap[photo.UserID])
	return &resp, nil
}

func (s *PhotoService) DeletePhoto(id, userID string) error {
	familyIDs := s.getFamilyIDs(userID)
	photo, err := s.photoRepo.FindByIDAndFamilyIDs(id, familyIDs)
	if err != nil {
		return errors.New(errPhotoNotFound)
	}

	// Remove file from disk if it's a local upload
	fileutil.RemoveUploadedFile(photo.ImageURL)

	return s.photoRepo.DeleteByFamily(id, familyIDs)
}

func (s *PhotoService) ToggleWall(id, userID string, req dto.ToggleWallRequest) (*dto.PhotoResponse, error) {
	familyIDs := s.getFamilyIDs(userID)
	photo, err := s.photoRepo.FindByIDAndFamilyIDs(id, familyIDs)
	if err != nil {
		return nil, errors.New(errPhotoNotFound)
	}

	if req.IsOnWall && !photo.IsOnWall {
		wallCount, countErr := s.photoRepo.CountWallPhotosByFamily(familyIDs)
		if countErr != nil {
			return nil, fmt.Errorf("failed to check wall count: %w", countErr)
		}
		if wallCount >= maxWallPhotos {
			return nil, fmt.Errorf("wall is full (max %d photos)", maxWallPhotos)
		}
	}

	photo.IsOnWall = req.IsOnWall
	photo.WallPosition = req.WallPosition
	if !req.IsOnWall {
		photo.WallPosition = nil
	}

	if err := s.photoRepo.Update(photo); err != nil {
		return nil, fmt.Errorf("failed to update photo: %w", err)
	}

	nicknameMap := s.buildNicknameMap(familyIDs)
	resp := toPhotoResponse(*photo, nicknameMap[photo.UserID])
	return &resp, nil
}

func (s *PhotoService) BatchUpdateWall(userID string, req dto.BatchWallUpdateRequest) ([]dto.PhotoResponse, error) {
	if len(req.Photos) > maxWallPhotos {
		return nil, fmt.Errorf("too many wall photos (max %d)", maxWallPhotos)
	}

	familyIDs := s.getFamilyIDs(userID)

	updates := make([]repository.WallUpdate, 0, len(req.Photos))
	for _, item := range req.Photos {
		updates = append(updates, repository.WallUpdate{
			PhotoID:  item.PhotoID,
			Position: item.Position,
		})
	}

	if err := s.photoRepo.BatchUpdateWallFamily(familyIDs, updates); err != nil {
		return nil, fmt.Errorf("failed to update wall: %w", err)
	}

	wallPhotos, err := s.photoRepo.FindWallPhotosByFamily(familyIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch wall photos: %w", err)
	}

	nicknameMap := s.buildNicknameMap(familyIDs)
	results := make([]dto.PhotoResponse, 0, len(wallPhotos))
	for _, p := range wallPhotos {
		results = append(results, toPhotoResponse(p, nicknameMap[p.UserID]))
	}
	return results, nil
}

func (s *PhotoService) downloadGeneratedImage(imgResp *openai.ImageResponse) (string, error) {
	return saveGeneratedImage(imgResp)
}

// saveGeneratedImage saves an AI-generated image response to disk and returns its URL path.
// Shared by PhotoService and task card generation.
func saveGeneratedImage(imgResp *openai.ImageResponse) (string, error) {
	uploadDir := "uploads/photos"
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create upload dir: %w", err)
	}

	filename := fmt.Sprintf("%s_%d.png", uuid.New().String(), time.Now().Unix())
	savePath := filepath.Join(uploadDir, filename)

	data := imgResp.Data[0]

	if data.URL != "" {
		return downloadFromURL(data.URL, savePath)
	}

	if data.B64JSON != "" {
		decoded, err := base64.StdEncoding.DecodeString(data.B64JSON)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 image: %w", err)
		}
		if err := os.WriteFile(savePath, decoded, 0o644); err != nil {
			return "", fmt.Errorf("failed to write image file: %w", err)
		}
		return "/uploads/photos/" + filename, nil
	}

	return "", fmt.Errorf("no image data in response")
}

// downloadFromURL fetches an image from a URL and saves it to savePath.
func downloadFromURL(imageURL, savePath string) (string, error) {
	// Validate URL to prevent SSRF
	if err := validateExternalURL(imageURL); err != nil {
		return "", fmt.Errorf("invalid image URL: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	out, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, io.LimitReader(resp.Body, 20<<20)); err != nil { // 20 MB max
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return "/uploads/photos/" + filepath.Base(savePath), nil
}

func toPhotoResponse(p model.Photo, ownerNickname string) dto.PhotoResponse {
	var tags []string
	if p.Tags != "" {
		_ = json.Unmarshal([]byte(p.Tags), &tags)
	}
	if tags == nil {
		tags = []string{}
	}

	return dto.PhotoResponse{
		ID:            p.ID,
		Title:         p.Title,
		Description:   p.Description,
		Tags:          tags,
		ImageURL:      p.ImageURL,
		IsOnWall:      p.IsOnWall,
		WallPosition:  p.WallPosition,
		Source:        p.Source,
		OwnerID:       p.UserID,
		OwnerNickname: ownerNickname,
		CreatedAt:     p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     p.UpdatedAt.Format(time.RFC3339),
	}
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen])
	}
	return s
}

// buildImagePrompt wraps the user's content description with style guidance.
// Rules:
//  1. Use sticker illustration style with white border and vector feel
//  2. Treat the user input as scene content, not a literal title
//  3. Default protagonist gender is based on user role: dad → male, otherwise female
func buildImagePrompt(userContent, userRole string) string {
	genderHint := "depict the person as a young woman"
	if userRole == "dad" {
		genderHint = "depict the person as a young man"
	}

	return fmt.Sprintf(
		"Sticker illustration: %s. "+
			"If the scene includes a person and no gender is specified, %s. "+
			"White border, vector style, high quality. "+
			"No text, no watermark, no signature.",
		userContent, genderHint,
	)
}

// validateExternalURL checks that a URL is safe to fetch (prevents SSRF).
func validateExternalURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL")
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	host := u.Hostname()

	// Block obvious internal hostnames
	if host == "localhost" || host == "127.0.0.1" || host == "::1" ||
		host == "0.0.0.0" || strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".internal") {
		return fmt.Errorf("internal host not allowed")
	}

	// Resolve and check IP ranges
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("DNS resolution failed: %w", err)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("internal IP not allowed")
		}
	}

	return nil
}
