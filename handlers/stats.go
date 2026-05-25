package handlers

import (
	"net/http"

	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	DB *database.Client
}

func NewStatsHandler(db *database.Client) *StatsHandler {
	return &StatsHandler{DB: db}
}

func (h *StatsHandler) Global(c *gin.Context) {
	stats, err := h.DB.GetGlobalStats(c.Request.Context())
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to get global stats")
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *StatsHandler) Subjects(c *gin.Context) {
	subjects, err := h.DB.ListSubjectsWithCounts(c.Request.Context())
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list subjects with counts")
		return
	}
	c.JSON(http.StatusOK, subjects)
}
