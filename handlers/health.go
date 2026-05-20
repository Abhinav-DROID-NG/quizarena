package handlers

import (
	"net/http"

	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	DB *database.Client
}

func NewHealthHandler(db *database.Client) *HealthHandler {
	return &HealthHandler{DB: db}
}

func (h *HealthHandler) Health(c *gin.Context) {
	status := gin.H{"status": "ok", "database": "ok"}
	if err := h.DB.Ping(c.Request.Context()); err != nil {
		status["status"] = "degraded"
		status["database"] = "down"
	}
	code := http.StatusOK
	if status["status"] != "ok" {
		code = http.StatusServiceUnavailable
	}
	c.JSON(code, status)
}
