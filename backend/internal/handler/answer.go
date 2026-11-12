package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/middleware"
	"github.com/momshell/backend/internal/service"
)

type AnswerHandler struct {
	communityService *service.CommunityService
	authService      *service.AuthService
	communityAI      *service.CommunityAIService
}

func NewAnswerHandler(communityService *service.CommunityService, authService *service.AuthService, communityAI *service.CommunityAIService) *AnswerHandler {
	return &AnswerHandler{communityService: communityService, authService: authService, communityAI: communityAI}
}

// GET /api/v1/community/questions/:id/answers
func (h *AnswerHandler) List(c *gin.Context) {
	questionID := c.Param("id")

	var params dto.AnswerListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	resp, err := h.communityService.GetAnswers(questionID, params, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// POST /api/v1/community/questions/:id/answers
func (h *AnswerHandler) Create(c *gin.Context) {
	questionID := c.Param("id")

	var req dto.AnswerCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	answer, err := h.communityService.CreateAnswer(questionID, req, user)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "问题不存在" {
			status = http.StatusNotFound
		} else if err.Error() == "仅认证专业人士可发布专家帖" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	// Trigger AI reply if answer mentions @小石光
	if h.communityAI != nil && h.communityAI.IsMentioned(req.Content) {
		go h.communityAI.HandleNewAnswer(answer.QuestionID, answer.ID)
	}

	c.JSON(http.StatusCreated, gin.H{"id": answer.ID, "status": string(answer.Status)})
}

// PUT /api/v1/community/answers/:id
func (h *AnswerHandler) Update(c *gin.Context) {
	answerID := c.Param("id")

	var req dto.AnswerUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := middleware.GetUserID(c)
	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	answer, err := h.communityService.UpdateAnswer(answerID, req, user)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "回答不存在" {
			status = http.StatusNotFound
		} else if err.Error() == "无权修改此回答" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": answer.ID, "status": string(answer.Status)})
}

// DELETE /api/v1/community/answers/:id
func (h *AnswerHandler) Delete(c *gin.Context) {
	answerID := c.Param("id")

	userID := middleware.GetUserID(c)
	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	if err := h.communityService.DeleteAnswer(answerID, user); err != nil {
		status := http.StatusBadRequest
		if err.Error() == "回答不存在" {
			status = http.StatusNotFound
		} else if err.Error() == "无权删除此回答" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
