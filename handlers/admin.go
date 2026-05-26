package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/Abhinav-DROID-NG/quizarena/middleware"
	"github.com/Abhinav-DROID-NG/quizarena/models"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	DB          *database.Client
	adminEmails map[string]struct{}
}

func NewAdminHandler(db *database.Client, emails []string) *AdminHandler {
	admins := make(map[string]struct{}, len(emails))
	for _, email := range emails {
		normalized := strings.ToLower(strings.TrimSpace(email))
		if normalized != "" {
			admins[normalized] = struct{}{}
		}
	}
	return &AdminHandler{DB: db, adminEmails: admins}
}

func (h *AdminHandler) ensureAdmin(c *gin.Context) bool {
	claimsRaw, ok := c.Get(middleware.UserClaimsKey)
	if !ok {
		utils.RespondError(c, http.StatusForbidden, "FORBIDDEN", "admin access required")
		return false
	}
	claims, ok := claimsRaw.(map[string]any)
	if !ok {
		utils.RespondError(c, http.StatusForbidden, "FORBIDDEN", "admin access required")
		return false
	}
	email, _ := claims["email"].(string)
	if _, ok = h.adminEmails[strings.ToLower(strings.TrimSpace(email))]; !ok {
		utils.RespondError(c, http.StatusForbidden, "FORBIDDEN", "admin access required")
		return false
	}
	return true
}

func (h *AdminHandler) AddQuestion(c *gin.Context) {
	if !h.ensureAdmin(c) {
		return
	}
	var req models.Question
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid payload")
		return
	}
	if err := utils.ValidateQuestionInput(req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid question constraints")
		return
	}
	id, err := h.DB.UpsertQuestion(c.Request.Context(), req)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to add question")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *AdminHandler) ListQuestions(c *gin.Context) {
	if !h.ensureAdmin(c) {
		return
	}
	questions, err := h.DB.ListQuestions(c.Request.Context(), 500)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list questions")
		return
	}
	c.JSON(http.StatusOK, questions)
}

func (h *AdminHandler) EditQuestion(c *gin.Context) {
	if !h.ensureAdmin(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	var req models.Question
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid payload")
		return
	}
	if err := utils.ValidateQuestionInput(req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid question constraints")
		return
	}
	req.ID = id
	_, err = h.DB.UpsertQuestion(c.Request.Context(), req)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to update question")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *AdminHandler) DeleteQuestion(c *gin.Context) {
	if !h.ensureAdmin(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	if err := h.DB.DeleteQuestion(c.Request.Context(), id); err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
