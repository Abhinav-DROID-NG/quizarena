package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Abhinav-DROID-NG/quizarena/cache"
	"github.com/Abhinav-DROID-NG/quizarena/config"
	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/Abhinav-DROID-NG/quizarena/handlers"
	"github.com/Abhinav-DROID-NG/quizarena/middleware"
	"github.com/Abhinav-DROID-NG/quizarena/services"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func main() {
	cfg := config.Load()
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	ctx := context.Background()
	db, err := database.New(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		logger.Fatal("failed to init db", zap.Error(err))
	}
	defer db.Close()

	redisStore := cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err := redisStore.Ping(ctx); err != nil {
		logger.Warn("redis unavailable, continuing without cache", zap.Error(err))
		redisStore = nil
	}

	tokenManager := utils.NewTokenManager(cfg.JWTSecret, cfg.JWTExpiration)
	eloEngine := services.NewEloEngine()

	var authRedisClient *redis.Client
	if redisStore != nil {
		authRedisClient = redisStore.Redis
	}
	authHandler := handlers.NewAuthHandler(db, tokenManager, authRedisClient, cfg.GoogleClientID)
	quizHandler := handlers.NewQuizHandler(db, eloEngine)
	leaderboardHandler := handlers.NewLeaderboardHandler(db, redisStore)
	userHandler := handlers.NewUserHandler(db)
	adminHandler := handlers.NewAdminHandler(db)
	healthHandler := handlers.NewHealthHandler(db, redisStore)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.RateLimitMiddleware(rate.Limit(10), 10))
	r.Use(middleware.CORS(config.CORSOrigins(cfg.FrontendOrigin)))

	r.GET("/health", healthHandler.Health)

	auth := r.Group("/auth")
	{
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
			logger.Fatal("server failed", zap.Error(err))
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
