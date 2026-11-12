package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/middleware"
	"github.com/momshell/backend/internal/service"
)

type ChatHandler struct {
	chatService *service.ChatService
}

func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

// POST /api/v1/companion/chat
func (h *ChatHandler) Chat(c *gin.Context) {
	var req dto.UserMessage
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	resp, err := h.chatService.Chat(c.Request.Context(), req, userID)
	if err != nil {
		respondWithAIError(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GET /api/v1/companion/profile
func (h *ChatHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	if userID != "" {
		profile, err := h.chatService.GetProfile(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, profile)
		return
	}

	// Guest - check session_id query param
	sessionID := c.Query("session_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "需要 session_id 或登录"})
		return
	}

	profile := h.chatService.GetGuestProfile(sessionID)
	c.JSON(http.StatusOK, profile)
}

// GET /api/v1/companion/memories
func (h *ChatHandler) GetMemories(c *gin.Context) {
	userID := middleware.GetUserID(c)
	resp, err := h.chatService.GetMemories(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/v1/companion/history
func (h *ChatHandler) GetHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	resp, err := h.chatService.GetConversationHistory(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// DELETE /api/v1/companion/history
func (h *ChatHandler) ClearHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if err := h.chatService.ClearConversationHistory(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "对话历史已清除"})
}

// DELETE /api/v1/companion/memories/:id
func (h *ChatHandler) DeleteMemory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	factID := c.Param("id")

	if err := h.chatService.DeleteMemory(userID, factID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}
