package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
)

const leaderboardQueryTimeout = 2 * time.Second

type LeaderboardHandler struct {
	DB *database.Client
}

func NewLeaderboardHandler(db *database.Client) *LeaderboardHandler {
	return &LeaderboardHandler{DB: db}
}

func (h *LeaderboardHandler) Global(c *gin.Context) {
	h.respond(c, "")
}

func (h *LeaderboardHandler) BySubject(c *gin.Context) {
	h.respond(c, c.Param("subject"))
}

func (h *LeaderboardHandler) respond(c *gin.Context, subject string) {
	if err := utils.ValidateSubject(subject); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid subject")
		return
	}
	limit := 100
	if l := c.Query("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil || parsed <= 0 || parsed > 500 {
			utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid leaderboard limit")
			return
		}
		limit = parsed
	}
	cursorElo, cursorUserID, err := parseLeaderboardCursor(c.Query("cursor"))
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid leaderboard cursor")
		return
	}
	queryCtx, cancel := context.WithTimeout(c.Request.Context(), leaderboardQueryTimeout)
	defer cancel()
	leaders, nextCursorElo, nextCursorUserID, hasMore, err := h.DB.ListLeaderboardPage(queryCtx, subject, limit, cursorElo, cursorUserID)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to load leaderboard")
		return
	}
	resp := gin.H{"items": leaders}
	if hasMore {
		resp["next_cursor"] = buildLeaderboardCursor(nextCursorElo, nextCursorUserID)
	}
	c.JSON(http.StatusOK, resp)
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
	queryCtx, cancel := context.WithTimeout(c.Request.Context(), leaderboardQueryTimeout)
	defer cancel()
	rank, err := h.DB.GetUserRank(queryCtx, uid)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to load leaderboard")
		return
	}
	c.JSON(http.StatusOK, gin.H{"rank": rank, "user": user})
}

func parseLeaderboardCursor(raw string) (int, int64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, 0, nil
	}
	parts := strings.Split(raw, ":")
	if len(parts) != 2 {
		return 0, 0, strconv.ErrSyntax
	}
	elo, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	uid, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return elo, uid, nil
}

func buildLeaderboardCursor(elo int, userID int64) string {
	return strconv.Itoa(elo) + ":" + strconv.FormatInt(userID, 10)
}
