package handlers

import (
	"net/http"

	"github.com/Abhinav-DROID-NG/quizarena/cache"
	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	DB    *database.Client
	Cache *cache.Client
}

func NewHealthHandler(db *database.Client, cacheClient *cache.Client) *HealthHandler {
	return &HealthHandler{DB: db, Cache: cacheClient}
}

func (h *HealthHandler) Health(c *gin.Context) {
	status := gin.H{"status": "ok", "database": "ok", "redis": "ok"}
	if err := h.DB.Ping(c.Request.Context()); err != nil {
		status["status"] = "degraded"
		status["database"] = "down"
	}
	if h.Cache != nil {
		if err := h.Cache.Ping(c.Request.Context()); err != nil {
			status["status"] = "degraded"
			status["redis"] = "down"
		}
	}
	code := http.StatusOK
	if status["status"] != "ok" {
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, status)
}
