package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/momshell/backend/internal/middleware"
	"github.com/momshell/backend/internal/service"
)

type ShellGiftHandler struct {
	shellGiftService *service.ShellGiftService
}

func NewShellGiftHandler(shellGiftService *service.ShellGiftService) *ShellGiftHandler {
	return &ShellGiftHandler{shellGiftService: shellGiftService}
}

// GET /api/v1/shell-gifts
func (h *ShellGiftHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	items, err := h.shellGiftService.ListForMom(userID)
	if err != nil {
		status := http.StatusBadRequest
		if err.Error() == "只有妈妈角色可以查看贝壳" {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, items)
}

// POST /api/v1/shell-gifts/:id/open
func (h *ShellGiftHandler) Open(c *gin.Context) {
	userID := middleware.GetUserID(c)
	giftID := c.Param("id")

	item, err := h.shellGiftService.OpenForMom(userID, giftID)
	if err != nil {
		status := http.StatusBadRequest
		switch err.Error() {
		case "贝壳不存在":
			status = http.StatusNotFound
		case "只有妈妈角色可以打开贝壳", "只能打开自己的贝壳":
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, item)
}
