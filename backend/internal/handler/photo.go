package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/middleware"
	"github.com/momshell/backend/internal/service"
)

const maxPhotoSize = 5 << 20 // 5 MB

var allowedPhotoTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

type PhotoHandler struct {
	photoService *service.PhotoService
}

func NewPhotoHandler(photoService *service.PhotoService) *PhotoHandler {
	return &PhotoHandler{photoService: photoService}
}

// GET /api/v1/photos
func (h *PhotoHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.photoService.ListPhotos(userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// POST /api/v1/photos/upload
func (h *PhotoHandler) Upload(c *gin.Context) {
	userID := middleware.GetUserID(c)

	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择要上传的照片"})
		return
	}
	defer func() { _ = file.Close() }()

	if header.Size > maxPhotoSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "照片大小不能超过 5MB"})
		return
	}

	contentType := header.Header.Get("Content-Type")
	if !allowedPhotoTypes[contentType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "仅支持 JPG、PNG、GIF、WebP 格式"})
		return
	}

	// Validate actual file content via magic bytes
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	detectedType := http.DetectContentType(buf[:n])
	if !allowedPhotoTypes[detectedType] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件内容与格式不匹配"})
		return
	}
	// Reset reader position for SaveUploadedFile
	if _, err := file.Seek(0, 0); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传失败"})
		return
	}

	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	}

	uploadDir := "uploads/photos"
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传失败"})
		return
	}

	filename := uuid.New().String() + ext
	savePath := filepath.Join(uploadDir, filename)

	if err := c.SaveUploadedFile(header, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "上传失败"})
		return
	}

	imageURL := fmt.Sprintf("/uploads/photos/%s", filename)
	title := c.PostForm("title")

	result, err := h.photoService.CreateFromUpload(userID, title, imageURL)
	if err != nil {
		_ = os.Remove(savePath)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, result)
}

// POST /api/v1/photos/generate
func (h *PhotoHandler) Generate(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.GeneratePhotoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Image generation may need async polling, allow up to 5 minutes
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	result, err := h.photoService.GeneratePhoto(ctx, userID, req)
	if err != nil {
		respondWithAIError(c, http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

// PUT /api/v1/photos/:id
func (h *PhotoHandler) Update(c *gin.Context) {
	userID := middleware.GetUserID(c)
	photoID := c.Param("id")

	var req dto.UpdatePhotoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.photoService.UpdatePhoto(photoID, userID, req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DELETE /api/v1/photos/:id
func (h *PhotoHandler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	photoID := c.Param("id")

	if err := h.photoService.DeletePhoto(photoID, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// PUT /api/v1/photos/:id/wall
func (h *PhotoHandler) ToggleWall(c *gin.Context) {
	userID := middleware.GetUserID(c)
	photoID := c.Param("id")

	var req dto.ToggleWallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.photoService.ToggleWall(photoID, userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// PUT /api/v1/photos/wall
func (h *PhotoHandler) BatchUpdateWall(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.BatchWallUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.photoService.BatchUpdateWall(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"photos": result})
}
