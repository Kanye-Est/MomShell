package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/momshell/backend/internal/dto"
	"github.com/momshell/backend/internal/middleware"
	"github.com/momshell/backend/internal/service"
)

type PerkCardHandler struct {
	perkCardService *service.PerkCardService
}

func NewPerkCardHandler(perkCardService *service.PerkCardService) *PerkCardHandler {
	return &PerkCardHandler{perkCardService: perkCardService}
}

// POST /api/v1/perk-cards
func (h *PerkCardHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req dto.CreatePerkCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.perkCardService.CreatePerkCard(userID, req)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "只有妈妈角色可以发放权益卡" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// GET /api/v1/perk-cards
func (h *PerkCardHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)

	items, err := h.perkCardService.GetPerkCards(userID)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "角色无权查看权益卡" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

// POST /api/v1/perk-cards/:id/use
func (h *PerkCardHandler) Use(c *gin.Context) {
	userID := middleware.GetUserID(c)
	cardID := c.Param("id")

	item, err := h.perkCardService.UsePerkCard(userID, cardID)
	if err != nil {
		status := http.StatusBadRequest
		switch err.Error() {
		case "权益卡不存在":
			status = http.StatusNotFound
		case "只有爸爸角色可以核销权益卡", "无权操作此权益卡":
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}
