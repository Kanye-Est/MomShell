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
	futureLetterRepo := repository.NewFutureLetterRepo(db)
	taskRepo := repository.NewTaskRepo(db)
	ragRepo := repository.NewRAGRepo(db)
	achievementRepo := repository.NewAchievementRepo(db)
	perkCardRepo := repository.NewPerkCardRepo(db)

	chatClient := openai.NewRuntimeClient(cfg.EmbeddingModel, func() (string, string, string) {
		settings := cfg.Runtime()
		return settings.OpenAIAPIKey, settings.OpenAIBaseURL, settings.OpenAIModel
	})
	if !chatClient.Enabled() {
		log.Println("[WARN] OPENAI_API_KEY not set. AI features require an API key.")
	}

	var ragService *service.RAGService
	if cfg.EmbeddingModel != "" && db.Migrator().HasTable(&model.KnowledgeEmbedding{}) {
		ragService = service.NewRAGService(chatClient, ragRepo, cfg)
	}

	// Initialize services
	moderationService := service.NewModerationService()
	authService := service.NewAuthService(cfg, userRepo)
	communityService := service.NewCommunityService(
		questionRepo, answerRepo, commentRepo,
		interactionRepo, tagRepo, userRepo,
		moderationService, ragService,
	)

	var firecrawlClient *firecrawl.Client
	if cfg.FirecrawlAPIKey != "" {
		firecrawlClient = firecrawl.NewClient(cfg.FirecrawlAPIKey)
	}

	chatService := service.NewChatService(chatClient, chatRepo, userRepo, ragService, firecrawlClient, cfg.JWTSecretKey)
	echoService := service.NewEchoService(chatClient, echoRepo, userRepo, ragService)
	photoService := service.NewPhotoService(photoRepo, userRepo, chatClient, cfg.ImageModel)
	whisperService := service.NewWhisperService(whisperRepo, futureLetterRepo, userRepo, chatRepo, taskRepo, chatClient, ragService)

	// Pre-generate daily letter when a mom logs in
	authService.OnLogin(func(userID string, role model.UserRole) {
		if role == model.RoleMom {
			whisperService.PreGenerateForUser(userID)
		}
	})

	achievementService := service.NewAchievementService(taskRepo, achievementRepo, userRepo)
	perkCardService := service.NewPerkCardService(perkCardRepo, userRepo)

	taskService := service.NewTaskService(taskRepo, userRepo, chatRepo, whisperRepo, photoRepo, chatClient, cfg.ImageModel, achievementService)

	// Ensure AI user exists for community AI replies
	aiUserID := ensureAIUser(userRepo)
	var communityAIService *service.CommunityAIService
	if aiUserID != "" {
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

	// Start background reindexing for RAG
	if ragService != nil && chatClient.Enabled() {
		ragService.BackgroundReindexAll(questionRepo, answerRepo, whisperRepo, echoRepo)
	}

	// Start background schedulers
	scheduler.StartPhotoCleanup(photoRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	questionHandler := handler.NewQuestionHandler(communityService, authService, communityAIService)
	answerHandler := handler.NewAnswerHandler(communityService, authService, communityAIService)
	commentHandler := handler.NewCommentHandler(communityService, authService, communityAIService)
	interactionHandler := handler.NewInteractionHandler(communityService)
	tagHandler := handler.NewTagHandler(communityService)
	chatHandler := handler.NewChatHandler(chatService)
	echoHandler := handler.NewEchoHandler(echoService, cfg.ImageModel != "")
	userHandler := handler.NewUserHandler(userService)
	adminHandler := handler.NewAdminHandler(adminService, authService)
	photoHandler := handler.NewPhotoHandler(photoService)
	whisperHandler := handler.NewWhisperHandler(whisperService)
	taskHandler := handler.NewTaskHandler(taskService, achievementService)
	perkCardHandler := handler.NewPerkCardHandler(perkCardService)

	// Setup Gin
	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("Invalid TRUSTED_PROXIES: %v", err)
	}
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
			PerkCard:    perkCardHandler,
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
