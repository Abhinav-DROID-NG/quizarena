package handlers

import (
	"net/http"
	"strconv"

	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/Abhinav-DROID-NG/quizarena/models"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	DB *database.Client
}

func NewAdminHandler(db *database.Client) *AdminHandler { return &AdminHandler{DB: db} }

func (h *AdminHandler) AddQuestion(c *gin.Context) {
	var req models.Question
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid payload")
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
	questions, err := h.DB.ListQuestions(c.Request.Context(), 500)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list questions")
		return
	}
	c.JSON(http.StatusOK, questions)
}

func (h *AdminHandler) EditQuestion(c *gin.Context) {
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
	req.ID = id
	_, err = h.DB.UpsertQuestion(c.Request.Context(), req)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to update question")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *AdminHandler) DeleteQuestion(c *gin.Context) {
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
