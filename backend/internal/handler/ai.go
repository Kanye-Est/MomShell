package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/momshell/backend/pkg/openai"
)

func respondWithAIError(c *gin.Context, status int, err error) {
	message := err.Error()
	for _, unavailable := range []error{openai.ErrNotConfigured, openai.ErrAuthentication, openai.ErrImageNotConfigured} {
		if errors.Is(err, unavailable) {
			status = http.StatusServiceUnavailable
			message = unavailable.Error()
			break
		}
	}
	c.JSON(status, gin.H{"error": message})
}
