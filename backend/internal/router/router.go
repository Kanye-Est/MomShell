package router

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/momshell/backend/internal/config"
	"github.com/momshell/backend/internal/handler"
	"github.com/momshell/backend/internal/middleware"
)

const (
	routeUsers  = "/users"
	routeUserID = "/users/:id"
)

// Handlers groups all handler dependencies for route setup.
type Handlers struct {
	Auth        *handler.AuthHandler
	Question    *handler.QuestionHandler
	Answer      *handler.AnswerHandler
	Comment     *handler.CommentHandler
	Interaction *handler.InteractionHandler
	Tag         *handler.TagHandler
	Chat        *handler.ChatHandler
	Echo        *handler.EchoHandler
	User        *handler.UserHandler
	Admin       *handler.AdminHandler
	Photo       *handler.PhotoHandler
	Whisper     *handler.WhisperHandler
	Task        *handler.TaskHandler
	ShellGift   *handler.ShellGiftHandler
}

func Setup(
	r *gin.Engine,
	cfg *config.Config,
	isAdmin middleware.AdminChecker,
	h *Handlers,
) {
	// Rate limiters
	authLimiter := middleware.RateLimit(10, 1*time.Minute)     // 10 req/min for auth
	aiLimiter := middleware.RateLimit(20, 1*time.Minute)       // 20 req/min for AI endpoints
	generalLimiter := middleware.RateLimit(120, 1*time.Minute) // 120 req/min general

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Serve uploaded files with security restrictions
	r.GET("/uploads/*filepath", secureStaticHandler("./uploads"))

	// Admin panel (HTML page, no auth required for serving the page)
	r.GET("/admin", h.Admin.ServeAdminPage)

	api := r.Group("/api/v1", generalLimiter)

	// ==================== Auth ====================
	auth := api.Group("/auth")
	{
		auth.POST("/register", authLimiter, h.Auth.Register)
		auth.POST("/login", authLimiter, h.Auth.Login)
		auth.POST("/refresh", authLimiter, h.Auth.Refresh)
		auth.POST("/forgot-password", authLimiter, h.Auth.ForgotPassword)
		auth.POST("/reset-password", authLimiter, h.Auth.ResetPassword)

		authRequired := auth.Group("", middleware.AuthRequired(cfg))
		{
			authRequired.POST("/change-password", h.Auth.ChangePassword)
			authRequired.GET("/me", h.Auth.GetMe)
			authRequired.PATCH("/me/role", h.Auth.UpdateRole)
			authRequired.PATCH("/me/tutorial", h.Auth.CompleteTutorial)
			authRequired.POST("/logout", h.Auth.Logout)
		}
	}

	// ==================== Community ====================
	community := api.Group("/community")
	{
		// Questions (optional auth for read, required for write)
		questions := community.Group("/questions")
		{
			questions.GET("", middleware.AuthOptional(cfg), h.Question.List)
			questions.GET("/hot", middleware.AuthOptional(cfg), h.Question.ListHot)
			questions.GET("/channel/:channel", middleware.AuthOptional(cfg), h.Question.ListByChannel)
			questions.GET("/:id", middleware.AuthOptional(cfg), h.Question.Get)

			questionsAuth := questions.Group("", middleware.AuthRequired(cfg))
			{
				questionsAuth.POST("", h.Question.Create)
				questionsAuth.PUT("/:id", h.Question.Update)
				questionsAuth.DELETE("/:id", h.Question.Delete)
			}

			// Answers under questions
			questions.GET("/:id/answers", middleware.AuthOptional(cfg), h.Answer.List)
			questions.POST("/:id/answers", middleware.AuthRequired(cfg), h.Answer.Create)
		}

		// Answers (update/delete by answer ID)
		answers := community.Group("/answers")
		{
			answers.PUT("/:id", middleware.AuthRequired(cfg), h.Answer.Update)
			answers.DELETE("/:id", middleware.AuthRequired(cfg), h.Answer.Delete)

			// Comments under answers
			answers.GET("/:id/comments", middleware.AuthOptional(cfg), h.Comment.List)
			answers.POST("/:id/comments", middleware.AuthRequired(cfg), h.Comment.Create)
		}

		// Comments (update/delete by comment ID)
		comments := community.Group("/comments")
		{
			comments.PUT("/:id", middleware.AuthRequired(cfg), h.Comment.Update)
			comments.DELETE("/:id", middleware.AuthRequired(cfg), h.Comment.Delete)
		}

		// Likes
		likes := community.Group("/likes", middleware.AuthRequired(cfg))
		{
			likes.POST("", h.Interaction.CreateLike)
			likes.DELETE("", h.Interaction.DeleteLike)
		}

		// Collections
		collections := community.Group("/collections", middleware.AuthRequired(cfg))
		{
			collections.POST("", h.Interaction.CreateCollection)
			collections.DELETE("/:id", h.Interaction.DeleteCollection)
			collections.GET("/my", h.Interaction.GetMyCollections)
		}

		// Tags
		tags := community.Group("/tags")
		{
			tags.GET("", h.Tag.List)
			tags.GET("/hot", h.Tag.ListHot)
			tags.POST("", middleware.AdminRequired(cfg, isAdmin), h.Tag.Create)
		}

		// User profile (community context)
		users := community.Group(routeUsers, middleware.AuthRequired(cfg))
		{
			users.GET("/me", h.User.GetMe)
			users.PUT("/me", h.User.UpdateMe)
			users.POST("/me/avatar", h.User.UploadAvatar)
			users.POST("/me/shell-code", h.User.GenerateShellCode)
			users.POST("/me/bind", h.User.BindPartner)
			users.DELETE("/me/bind", h.User.UnbindPartner)
			users.GET("/me/questions", h.User.GetMyQuestions)
			users.GET("/me/answers", h.User.GetMyAnswers)
		}
	}

	// ==================== Companion (AI Chat) ====================
	companion := api.Group("/companion")
	{
		companion.POST("/chat", aiLimiter, middleware.AuthOptional(cfg), h.Chat.Chat)
		companion.GET("/profile", middleware.AuthOptional(cfg), h.Chat.GetProfile)
		companion.GET("/memories", middleware.AuthRequired(cfg), h.Chat.GetMemories)
		companion.DELETE("/memories/:id", middleware.AuthRequired(cfg), h.Chat.DeleteMemory)
		companion.GET("/history", middleware.AuthRequired(cfg), h.Chat.GetHistory)
		companion.DELETE("/history", middleware.AuthRequired(cfg), h.Chat.ClearHistory)
	}

	echo := api.Group("/echo", middleware.AuthRequired(cfg))
	{
		echo.GET("/identity-tags", h.Echo.GetIdentityTags)
		echo.POST("/identity-tags", h.Echo.CreateIdentityTag)
		echo.DELETE("/identity-tags/:id", h.Echo.DeleteIdentityTag)

		echo.GET("/memoirs", h.Echo.GetMemoirs)
		echo.POST("/memoirs/generate", aiLimiter, h.Echo.GenerateMemoir)
		echo.POST("/memoirs/:id/rate", h.Echo.RateMemoir)
	}

	// ==================== Photos ====================
	photos := api.Group("/photos", middleware.AuthRequired(cfg))
	{
		photos.GET("", h.Photo.List)
		photos.POST("/upload", h.Photo.Upload)
		photos.POST("/generate", aiLimiter, h.Photo.Generate)
		photos.PUT("/wall", h.Photo.BatchUpdateWall)
		photos.PUT("/:id", h.Photo.Update)
		photos.DELETE("/:id", h.Photo.Delete)
		photos.PUT("/:id/wall", h.Photo.ToggleWall)
	}

	// ==================== Whisper (Heart Words) ====================
	whisper := api.Group("/whisper", middleware.AuthRequired(cfg))
	{
		whisper.POST("", h.Whisper.Create)
		whisper.GET("", h.Whisper.List)
		whisper.GET("/tips", aiLimiter, h.Whisper.Tips)
	}

	// ==================== Tasks ====================
	tasks := api.Group("/tasks", middleware.AuthRequired(cfg))
	{
		tasks.GET("/daily", h.Task.DailyTasks)
		tasks.POST("/:id/complete", h.Task.Complete)
		tasks.GET("/partner", h.Task.PartnerTasks)
		tasks.POST("/:id/score", h.Task.Score)
		tasks.POST("/:id/reject", h.Task.Reject)
		tasks.GET("/stats", h.Task.Stats)
		tasks.GET("/baby-age", h.Task.GetBabyAge)
		tasks.PUT("/baby-age", h.Task.SetBabyAge)
	}

	// ==================== Shell Gifts ====================
	shellGifts := api.Group("/shell-gifts", middleware.AuthRequired(cfg))
	{
		shellGifts.GET("", h.ShellGift.List)
		shellGifts.POST("/:id/open", h.ShellGift.Open)
	}

	// ==================== Admin ====================
	adminAPI := api.Group("/admin", middleware.AdminRequired(cfg, isAdmin))
	{
		adminAPI.GET("/stats", h.Admin.GetStats)
		adminAPI.GET(routeUsers, h.Admin.ListUsers)
		adminAPI.GET(routeUserID, h.Admin.GetUser)
		adminAPI.POST(routeUsers, h.Admin.CreateUser)
		adminAPI.PATCH(routeUserID, h.Admin.UpdateUser)
		adminAPI.DELETE(routeUserID, h.Admin.DeleteUser)
		adminAPI.GET("/config", h.Admin.GetConfig)
		adminAPI.PATCH("/config", h.Admin.UpdateConfig)
		adminAPI.GET("/photos", h.Admin.ListPhotos)
		adminAPI.DELETE("/photos/:id", h.Admin.DeletePhoto)
	}
}

// allowedImageExts defines image extensions that are safe to serve inline.
var allowedImageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

// blockedExts defines file extensions that must never be served.
var blockedExts = map[string]bool{
	".svg":  true,
	".html": true,
	".htm":  true,
	".xml":  true,
	".js":   true,
	".css":  true,
}

// secureStaticHandler returns a handler that serves files from the given root
// directory with security headers and file type restrictions.
func secureStaticHandler(root string) gin.HandlerFunc {
	fs := http.Dir(root)
	fileServer := http.StripPrefix("/uploads", http.FileServer(fs))

	return func(c *gin.Context) {
		// Clean the path to prevent traversal attacks
		reqPath := filepath.Clean(c.Param("filepath"))

		// Ensure the cleaned path does not escape the root
		if strings.Contains(reqPath, "..") {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		ext := strings.ToLower(filepath.Ext(reqPath))

		// Block dangerous file types
		if blockedExts[ext] {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// Set security headers
		c.Header("X-Content-Type-Options", "nosniff")

		if allowedImageExts[ext] {
			c.Header("Content-Disposition", "inline")
		}

		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}
