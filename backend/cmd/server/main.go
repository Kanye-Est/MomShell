package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/momshell/backend/internal/config"
	"github.com/momshell/backend/internal/database"
	"github.com/momshell/backend/internal/handler"
	"github.com/momshell/backend/internal/middleware"
	"github.com/momshell/backend/internal/model"
	"github.com/momshell/backend/internal/repository"
	"github.com/momshell/backend/internal/router"
	"github.com/momshell/backend/internal/scheduler"
	"github.com/momshell/backend/internal/service"
	"github.com/momshell/backend/pkg/firecrawl"
	"github.com/momshell/backend/pkg/openai"
	"github.com/momshell/backend/pkg/password"
)

func main() {
	// Load config
	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Connected to database")

	// Run migrations
	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Create initial admin if configured
	createInitialAdmin(cfg, repository.NewUserRepo(db))

	// Initialize repositories
	userRepo := repository.NewUserRepo(db)
	questionRepo := repository.NewQuestionRepo(db)
	answerRepo := repository.NewAnswerRepo(db)
	commentRepo := repository.NewCommentRepo(db)
	interactionRepo := repository.NewInteractionRepo(db)
	tagRepo := repository.NewTagRepo(db)
	chatRepo := repository.NewChatRepo(db)
	echoRepo := repository.NewEchoRepo(db)
	photoRepo := repository.NewPhotoRepo(db)
	whisperRepo := repository.NewWhisperRepo(db)
	taskRepo := repository.NewTaskRepo(db)
	shellGiftRepo := repository.NewShellGiftRepo(db)

	// Initialize services
	moderationService := service.NewModerationService()
	authService := service.NewAuthService(cfg, userRepo)
	communityService := service.NewCommunityService(
		questionRepo, answerRepo, commentRepo,
		interactionRepo, tagRepo, userRepo,
		moderationService,
	)

	var chatClient *openai.Client
	if cfg.OpenAIAPIKey != "" {
		chatClient = openai.NewClient(cfg.OpenAIAPIKey, cfg.OpenAIBaseURL, cfg.OpenAIModel)
	} else {
		log.Println("[WARN] OPENAI_API_KEY not set, chat and AI task generation will not work")
		chatClient = openai.NewClient("dummy", cfg.OpenAIBaseURL, cfg.OpenAIModel)
	}

	var firecrawlClient *firecrawl.Client
	if cfg.FirecrawlAPIKey != "" {
		firecrawlClient = firecrawl.NewClient(cfg.FirecrawlAPIKey)
	}

	chatService := service.NewChatService(chatClient, chatRepo, userRepo, firecrawlClient, cfg.JWTSecretKey)
	echoService := service.NewEchoService(chatClient, echoRepo, userRepo)
	photoService := service.NewPhotoService(photoRepo, userRepo, chatClient, cfg.ImageModel)
	whisperService := service.NewWhisperService(whisperRepo, userRepo, chatClient)

	// Pass nil to shell gift service when no real API key — avoids dummy client network calls
	var shellGiftAIClient *openai.Client
	if cfg.OpenAIAPIKey != "" {
		shellGiftAIClient = chatClient
	}
	shellGiftService := service.NewShellGiftService(shellGiftRepo, userRepo, shellGiftAIClient)

	// Pass nil to task service when no real API key — avoids dummy client network calls
	var taskAIClient *openai.Client
	if cfg.OpenAIAPIKey != "" {
		taskAIClient = chatClient
	}
	taskService := service.NewTaskService(taskRepo, userRepo, chatRepo, taskAIClient, shellGiftService)

	// Ensure AI user exists for community AI replies
	aiUserID := ensureAIUser(userRepo)
	var communityAIService *service.CommunityAIService
	if aiUserID != "" && cfg.OpenAIAPIKey != "" {
		communityAIService = service.NewCommunityAIService(
			chatClient, firecrawlClient,
			questionRepo, answerRepo, commentRepo,
			userRepo, aiUserID,
		)
	}

	userService := service.NewUserService(
		db, userRepo, questionRepo, answerRepo,
		interactionRepo, communityService,
	)

	// Initialize admin layer
	adminRepo := repository.NewAdminRepo(db)
	adminService := service.NewAdminService(cfg, adminRepo, userRepo, photoRepo)

	// Start background schedulers
	scheduler.StartPhotoCleanup(photoRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService, cfg)
	questionHandler := handler.NewQuestionHandler(communityService, authService, communityAIService)
	answerHandler := handler.NewAnswerHandler(communityService, authService, communityAIService)
	commentHandler := handler.NewCommentHandler(communityService, authService, communityAIService)
	interactionHandler := handler.NewInteractionHandler(communityService)
	tagHandler := handler.NewTagHandler(communityService)
	chatHandler := handler.NewChatHandler(chatService)
	echoHandler := handler.NewEchoHandler(echoService)
	userHandler := handler.NewUserHandler(userService)
	adminHandler := handler.NewAdminHandler(adminService, authService)
	photoHandler := handler.NewPhotoHandler(photoService)
	whisperHandler := handler.NewWhisperHandler(whisperService)
	taskHandler := handler.NewTaskHandler(taskService)
	shellGiftHandler := handler.NewShellGiftHandler(shellGiftService)

	// Setup Gin
	r := gin.New()
	_ = r.SetTrustedProxies(nil) // Don't trust any proxy headers by default
	r.Use(middleware.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg))
	r.Use(gin.Logger())

	// Register routes
	router.Setup(
		r, cfg,
		adminHandler.IsAdmin,
		&router.Handlers{
			Auth:        authHandler,
			Question:    questionHandler,
			Answer:      answerHandler,
			Comment:     commentHandler,
			Interaction: interactionHandler,
			Tag:         tagHandler,
			Chat:        chatHandler,
			Echo:        echoHandler,
			User:        userHandler,
			Admin:       adminHandler,
			Photo:       photoHandler,
			Whisper:     whisperHandler,
			Task:        taskHandler,
			ShellGift:   shellGiftHandler,
		},
	)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting server on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func createInitialAdmin(cfg *config.Config, userRepo *repository.UserRepo) {
	if cfg.AdminUsername == "" || cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		return
	}

	exists, _ := userRepo.ExistsByUsernameOrEmail(cfg.AdminUsername, cfg.AdminEmail)
	if exists {
		log.Println("Admin user already exists, skipping creation")
		return
	}

	hash, err := password.Hash(cfg.AdminPassword)
	if err != nil {
		log.Printf("Failed to hash admin password: %v", err)
		return
	}

	admin := &model.User{
		Username:     cfg.AdminUsername,
		Email:        cfg.AdminEmail,
		PasswordHash: hash,
		Nickname:     "Admin",
		Role:         model.RoleMom,
		IsAdmin:      true,
		IsActive:     true,
	}

	if err := userRepo.Create(admin); err != nil {
		log.Printf("Failed to create admin user: %v", err)
		return
	}

	log.Printf("Admin user created: %s", cfg.AdminUsername)
}

func ensureAIUser(userRepo *repository.UserRepo) string {
	aiUsername := os.Getenv("AI_USER_USERNAME")
	if aiUsername == "" {
		aiUsername = "xiaoshiguang"
		log.Println("[WARN] AI_USER_USERNAME not set, using default: xiaoshiguang")
	}
	aiPasswd := os.Getenv("AI_USER_PASSWORD")
	if aiPasswd == "" {
		// Generate a random password; AI user never logs in interactively.
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			log.Fatalf("failed to generate random AI user password: %v", err)
		}
		aiPasswd = hex.EncodeToString(b)
		log.Println("[WARN] AI_USER_PASSWORD not set, using random password")
	}

	user, err := userRepo.FindByUsernameOrEmail(aiUsername)
	if err == nil {
		return user.ID
	}

	hash, err := password.Hash(aiPasswd)
	if err != nil {
		log.Printf("Failed to hash AI user password: %v", err)
		return ""
	}

	aiUser := &model.User{
		Username:     aiUsername,
		Email:        "ai@momshell.com",
		PasswordHash: hash,
		Nickname:     "小石光",
		Role:         model.RoleAIAssistant,
		IsActive:     true,
	}

	if err := userRepo.Create(aiUser); err != nil {
		log.Printf("Failed to create AI user: %v", err)
		return ""
	}

	log.Printf("AI user created: %s (ID: %s)", aiUser.Username, aiUser.ID)
	return aiUser.ID
}
