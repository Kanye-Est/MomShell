package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/middleware"
	"github.com/momshell/backend/internal/service"
)

type TaskHandler struct {
	taskService        *service.TaskService
	achievementService *service.AchievementService
}

func NewTaskHandler(taskService *service.TaskService, achievementService *service.AchievementService) *TaskHandler {
	return &TaskHandler{
		taskService:        taskService,
		achievementService: achievementService,
	}
}

// GET /api/v1/tasks/daily
func (h *TaskHandler) DailyTasks(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.taskService.GetDailyTasks(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

// POST /api/v1/tasks/:id/complete
func (h *TaskHandler) Complete(c *gin.Context) {
	taskID := c.Param("id")
	userID := middleware.GetUserID(c)

	var req dto.CompleteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.taskService.CompleteTask(userID, taskID, req.ProofPhotoURL)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "任务不存在" {
			status = http.StatusNotFound
		} else if err.Error() == "无权操作此任务" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

// GET /api/v1/tasks/partner
func (h *TaskHandler) PartnerTasks(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.taskService.GetPartnerTasks(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

// POST /api/v1/tasks/:id/score
func (h *TaskHandler) Score(c *gin.Context) {
	taskID := c.Param("id")
	userID := middleware.GetUserID(c)

	var req dto.TaskScore
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.taskService.ScoreTask(userID, taskID, req)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "任务不存在" {
			status = http.StatusNotFound
		} else if err.Error() == "只能验收伴侣的任务" || err.Error() == "只有妈妈角色可以验收任务" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

// POST /api/v1/tasks/:id/reject
func (h *TaskHandler) Reject(c *gin.Context) {
	taskID := c.Param("id")
	userID := middleware.GetUserID(c)

	var req struct {
		Comment string `json:"comment" binding:"max=500"`
	}
	_ = c.ShouldBindJSON(&req)

	item, err := h.taskService.RejectTask(userID, taskID, req.Comment)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "任务不存在" {
			status = http.StatusNotFound
		} else if err.Error() == "只能验收伴侣的任务" || err.Error() == "只有妈妈角色可以验收任务" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}

// GET /api/v1/tasks/stats
func (h *TaskHandler) Stats(c *gin.Context) {
	userID := middleware.GetUserID(c)
	stats, err := h.taskService.GetTaskStats(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GET /api/v1/tasks/baby-age
func (h *TaskHandler) GetBabyAge(c *gin.Context) {
	userID := middleware.GetUserID(c)
	resp, err := h.taskService.GetBabyAge(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// PUT /api/v1/tasks/baby-age
func (h *TaskHandler) SetBabyAge(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.SetBabyAgeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.taskService.SetBabyAge(userID, req.AgeStage); err != nil {
		status := http.StatusInternalServerError
		if err.Error() == "用户不存在" {
			status = http.StatusNotFound
		} else if err.Error() != "保存失败，请重试" {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// GET /api/v1/tasks/radar
func (h *TaskHandler) Radar(c *gin.Context) {
	userID := middleware.GetUserID(c)
	radar, err := h.achievementService.GetSkillRadar(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, radar)
}

// GET /api/v1/tasks/achievements
func (h *TaskHandler) Achievements(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.achievementService.GetAchievements(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}
