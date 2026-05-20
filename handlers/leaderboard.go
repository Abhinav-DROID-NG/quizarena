package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Abhinav-DROID-NG/quizarena/cache"
	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
)

type LeaderboardHandler struct {
	DB    *database.Client
	Cache *cache.Client
}

func NewLeaderboardHandler(db *database.Client, cacheClient *cache.Client) *LeaderboardHandler {
	return &LeaderboardHandler{DB: db, Cache: cacheClient}
}

func (h *LeaderboardHandler) Global(c *gin.Context) {
	h.respond(c, "")
}

func (h *LeaderboardHandler) BySubject(c *gin.Context) {
	h.respond(c, c.Param("subject"))
}

func (h *LeaderboardHandler) respond(c *gin.Context, subject string) {
	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}
	cacheKey := fmt.Sprintf("leaderboard:%s:%d", subject, limit)
	if h.Cache != nil {
		var cached []any
		if found, err := h.Cache.GetJSON(c.Request.Context(), cacheKey, &cached); err == nil && found {
			c.JSON(http.StatusOK, cached)
			return
		}
	}
	leaders, err := h.DB.ListLeaderboard(c.Request.Context(), subject, limit)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to load leaderboard")
		return
	}
	if h.Cache != nil {
		_ = h.Cache.SetJSON(c.Request.Context(), cacheKey, leaders, 30*time.Second)
	}
	c.JSON(http.StatusOK, leaders)
}

func (h *LeaderboardHandler) UserRank(c *gin.Context) {
	uid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid user id")
		return
	}
	user, err := h.DB.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	leaders, err := h.DB.ListLeaderboard(c.Request.Context(), "", 10000)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to load leaderboard")
		return
	}
	rank := 0
	for i, leader := range leaders {
		if leader.ID == uid {
			rank = i + 1
			break
		}
	}
	c.JSON(http.StatusOK, gin.H{"rank": rank, "user": user})
}
