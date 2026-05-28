package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Abhinav-DROID-NG/quizarena/config"
	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/Abhinav-DROID-NG/quizarena/handlers"
	"github.com/Abhinav-DROID-NG/quizarena/middleware"
	"github.com/Abhinav-DROID-NG/quizarena/services"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func main() {
	// Runtime configuration is loaded from environment variables (and optional .env):
	// PORT, FRONTEND_ORIGIN, JWT_SECRET, JWT_EXP_HOURS, GOOGLE_CLIENT_ID,
	// ADMIN_EMAILS, DATABASE_URL, DB_MAX_CONNS, SHUTDOWN_TIMEOUT_SEC.
	cfg := config.Load()
	logger, err := zap.NewProduction()
	if err != nil {
		log.Printf("failed to initialize production logger: %v", err)
		logger = zap.NewNop()
	}
	defer func() { _ = logger.Sync() }()
	if warnings, cfgErr := config.Validate(cfg); cfgErr != nil {
		logger.Fatal("invalid configuration", zap.Error(cfgErr))
	} else {
		for _, warning := range warnings {
			logger.Warn("configuration warning", zap.String("warning", warning))
		}
	}

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelStartup()
	db, err := database.New(startupCtx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		logger.Fatal("failed to init db", zap.Error(err))
	}
	if err := db.Ping(startupCtx); err != nil {
		logger.Fatal("database ping failed", zap.Error(err))
	}
	defer db.Close()

	tokenManager := utils.NewTokenManager(cfg.JWTSecret, cfg.JWTExpiration)
	eloEngine := services.NewEloEngine()

	authHandler := handlers.NewAuthHandler(db, tokenManager, cfg.GoogleClientID)
	quizHandler := handlers.NewQuizHandler(db, eloEngine)
	leaderboardHandler := handlers.NewLeaderboardHandler(db)
	userHandler := handlers.NewUserHandler(db)
	adminHandler := handlers.NewAdminHandler(db, cfg.AdminEmails)
	healthHandler := handlers.NewHealthHandler(db)
	statsHandler := handlers.NewStatsHandler(db)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.RateLimitMiddleware(rate.Limit(10), 10))
	r.Use(middleware.CORSWithLogger(config.CORSOrigins(cfg.FrontendOrigin), logger))

	frontendDir := filepath.Clean("./frontend")
	frontendHTML := filepath.Clean("./frontend/quizarena-production.html")
	frontendReady := true
	if dirInfo, statErr := os.Stat(frontendDir); statErr != nil || !dirInfo.IsDir() {
		frontendReady = false
		logger.Error("frontend directory unavailable", zap.String("path", frontendDir), zap.Error(statErr))
	}
	if fileInfo, statErr := os.Stat(frontendHTML); statErr != nil || fileInfo.IsDir() {
		frontendReady = false
		logger.Error("frontend html unavailable", zap.String("path", frontendHTML), zap.Error(statErr))
	}

	r.GET("/", func(c *gin.Context) {
		if !frontendReady {
			utils.RespondError(c, http.StatusServiceUnavailable, "FRONTEND_UNAVAILABLE", "frontend not available")
			return
		}
		c.File(frontendHTML)
	})
	r.GET("/app-config", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"google_client_id": cfg.GoogleClientID,
		})
	})
	if frontendReady {
		r.Static("/frontend", frontendDir)
	}
	r.GET("/health", healthHandler.Health)
	r.GET("/stats", statsHandler.Global)
	r.GET("/subjects-detailed", statsHandler.Subjects)

	auth := r.Group("/auth")
	{
		auth.POST("/register", middleware.AuthRateLimitMiddleware(rate.Limit(1), 5), authHandler.Register)
		auth.POST("/login", middleware.AuthRateLimitMiddleware(rate.Limit(1), 5), authHandler.Login)
		auth.POST("/google", authHandler.GoogleAuth)
		auth.POST("/logout", authHandler.Logout)
		auth.GET("/me", middleware.JWTAuth(tokenManager), authHandler.Me)
	}

	quiz := r.Group("/quiz", middleware.JWTAuth(tokenManager))
	{
		quiz.POST("/start", quizHandler.Start)
		quiz.GET("/next-question", quizHandler.NextQuestion)
		quiz.POST("/submit-answer", quizHandler.SubmitAnswer)
		quiz.GET("/session/:id", quizHandler.SessionHistory)
	}

	r.GET("/leaderboard", leaderboardHandler.Global)
	r.GET("/leaderboard/subject/:subject", leaderboardHandler.BySubject)
	r.GET("/leaderboard/user/:id", leaderboardHandler.UserRank)
	r.GET("/subjects", quizHandler.ListSubjects)

	user := r.Group("/user", middleware.JWTAuth(tokenManager))
	{
		user.GET("/profile", userHandler.Profile)
		user.GET("/dashboard", userHandler.Dashboard)
		user.PUT("/profile", userHandler.UpdateProfile)
	}

	admin := r.Group("/admin", middleware.JWTAuth(tokenManager))
	{
		admin.POST("/questions", adminHandler.AddQuestion)
		admin.GET("/questions", adminHandler.ListQuestions)
		admin.PUT("/questions/:id", adminHandler.EditQuestion)
		admin.DELETE("/questions/:id", adminHandler.DeleteQuestion)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("starting server", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutSec)*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}
}
