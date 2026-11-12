package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/middleware"
	"github.com/momshell/backend/internal/service"
	"gorm.io/gorm"
)

type EchoHandler struct {
	echoService            *service.EchoService
	imageGenerationEnabled bool
}

func NewEchoHandler(echoService *service.EchoService, imageGenerationEnabled bool) *EchoHandler {
	return &EchoHandler{echoService: echoService, imageGenerationEnabled: imageGenerationEnabled}
}

func (h *EchoHandler) GetIdentityTags(c *gin.Context) {
	userID := middleware.GetUserID(c)

	resp, err := h.echoService.GetIdentityTags(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *EchoHandler) CreateIdentityTag(c *gin.Context) {
	var req dto.IdentityTagCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	tag, err := h.echoService.CreateIdentityTag(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tag)
}

func (h *EchoHandler) DeleteIdentityTag(c *gin.Context) {
	userID := middleware.GetUserID(c)
	tagID := c.Param("id")

	err := h.echoService.DeleteIdentityTag(userID, tagID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "identity tag not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *EchoHandler) GetMemoirs(c *gin.Context) {
	limit := 10
	offset := 0

	if limitRaw := c.Query("limit"); limitRaw != "" {
		parsed, err := strconv.Atoi(limitRaw)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
			return
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = parsed
	}

	if offsetRaw := c.Query("offset"); offsetRaw != "" {
		parsed, err := strconv.Atoi(offsetRaw)
		if err != nil || parsed < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
			return
		}
		offset = parsed
	}

	userID := middleware.GetUserID(c)
	resp, err := h.echoService.GetMemoirs(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *EchoHandler) GenerateMemoir(c *gin.Context) {
	var req dto.GenerateMemoirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	memoir, err := h.echoService.GenerateMemoir(c.Request.Context(), userID, req)
	if err != nil {
		respondWithAIError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, dto.GenerateMemoirResponse{
		Memoir:                 memoir,
		ImageGenerationEnabled: h.imageGenerationEnabled,
	})
}

func (h *EchoHandler) RateMemoir(c *gin.Context) {
	var req dto.RateMemoirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	memoirID := c.Param("id")

	memoir, err := h.echoService.RateMemoir(userID, memoirID, req.Rating)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "memoir not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, memoir)
}
