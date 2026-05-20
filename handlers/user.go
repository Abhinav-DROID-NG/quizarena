package handlers

import (
	"net/http"

	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/Abhinav-DROID-NG/quizarena/middleware"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	DB *database.Client
}

func NewUserHandler(db *database.Client) *UserHandler { return &UserHandler{DB: db} }

func userIDFromCtx(c *gin.Context) (int64, bool) {
	uidRaw, ok := c.Get(middleware.UserIDKey)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user")
		return 0, false
	}
	uid, ok := uidRaw.(int64)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user")
		return 0, false
	}
	return uid, true
}

func (h *UserHandler) Profile(c *gin.Context) {
	uid, ok := userIDFromCtx(c)
	if !ok {
		return
	}
	user, err := h.DB.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) Dashboard(c *gin.Context) {
	uid, ok := userIDFromCtx(c)
	if !ok {
		return
	}
	user, err := h.DB.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"current_elo":            user.CurrentElo,
		"peak_elo":               user.PeakElo,
		"accuracy_percentage":    user.AccuracyPercentage,
		"average_response_time":  user.AverageResponseTime,
		"total_questions_solved": user.TotalQuestions,
	})
}

type UpdateProfileRequest struct {
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	uid, ok := userIDFromCtx(c)
	if !ok {
		return
	}
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid payload")
		return
	}
	_, err := h.DB.Pool.Exec(c.Request.Context(), `UPDATE users SET name = $1, picture = $2, updated_at = NOW() WHERE id = $3`, req.Name, req.Picture, uid)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to update profile")
		return
	}
	user, err := h.DB.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	c.JSON(http.StatusOK, user)
}
