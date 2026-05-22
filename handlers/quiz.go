package handlers

import (
	"net/http"
	"strconv"

	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/Abhinav-DROID-NG/quizarena/middleware"
	"github.com/Abhinav-DROID-NG/quizarena/models"
	"github.com/Abhinav-DROID-NG/quizarena/services"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
)

type QuizHandler struct {
	DB  *database.Client
	Elo *services.EloEngine
}

func NewQuizHandler(db *database.Client, elo *services.EloEngine) *QuizHandler {
	return &QuizHandler{DB: db, Elo: elo}
}

type StartQuizRequest struct {
	Subject string `json:"subject"`
}

type SubmitAnswerRequest struct {
	SessionID    int64   `json:"session_id" binding:"required"`
	QuestionID   int64   `json:"question_id" binding:"required"`
	Selected     string  `json:"selected_answer" binding:"required"`
	TimeTakenSec float64 `json:"time_taken_seconds" binding:"required"`
	SkipsInRow   int     `json:"skips_in_row"`
	WrongInRow   int     `json:"wrong_in_row"`
}

func getUserID(c *gin.Context) (int64, bool) {
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

func (h *QuizHandler) Start(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var req StartQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid payload")
		return
	}
	sid, err := h.DB.CreateSession(c.Request.Context(), uid, req.Subject)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to start session")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"session_id": sid})
}

func (h *QuizHandler) NextQuestion(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	user, err := h.DB.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	subject := c.Query("subject")
	target := h.Elo.NextTargetElo(user.CurrentElo)
	question, err := h.DB.GetAdaptiveQuestion(c.Request.Context(), subject, target)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "no question available")
		return
	}
	c.JSON(http.StatusOK, question)
}

func (h *QuizHandler) SubmitAnswer(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	var req SubmitAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid payload")
		return
	}
	_, err := h.DB.GetSession(c.Request.Context(), req.SessionID, uid)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "session not found")
		return
	}
	question, err := h.DB.GetQuestionByID(c.Request.Context(), req.QuestionID)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "question not found")
		return
	}
	user, err := h.DB.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	correct := req.Selected == question.CorrectAnswer
	timeScore := h.Elo.TimeScore(req.TimeTakenSec, question.ExpectedTimeSeconds)
	performance := h.Elo.PerformanceScore(timeScore, correct)
	_, eloChange := h.Elo.CalculateNewElo(user.CurrentElo, question.QuestionElo, question.Difficulty, performance)
	eloChange = h.Elo.ApplyAntiGuessingPenalty(eloChange, correct, req.TimeTakenSec, question.ExpectedTimeSeconds, req.SkipsInRow)
	newElo := user.CurrentElo + eloChange
	nextTarget := h.Elo.NextTargetElo(newElo)
	nextDiff := h.Elo.NextQuestionDifficulty(newElo, nextTarget)
	confidence := h.Elo.ConfidenceScore(correct, timeScore, req.WrongInRow, req.SkipsInRow)

	if err := h.DB.SaveAnswerAndUpdateStats(c.Request.Context(), req.SessionID, req.QuestionID, uid, req.Selected, question.CorrectAnswer, req.TimeTakenSec, timeScore, performance, eloChange, newElo); err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to persist answer")
		return
	}

	resp := models.AnswerResponse{
		Correct:                correct,
		CorrectAnswer:          question.CorrectAnswer,
		TimeTaken:              req.TimeTakenSec,
		TimeScore:              timeScore,
		PerformanceScore:       performance,
		EloChange:              eloChange,
		NewUserElo:             newElo,
		NextQuestionDifficulty: string(nextDiff),
		ConfidenceScore:        confidence,
	}
	c.JSON(http.StatusOK, resp)
}

func (h *QuizHandler) SessionHistory(c *gin.Context) {
	uid, ok := getUserID(c)
	if !ok {
		return
	}
	sidStr := c.Param("id")
	sid, err := strconv.ParseInt(sidStr, 10, 64)
	if err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid session id")
		return
	}
	session, err := h.DB.GetSession(c.Request.Context(), sid, uid)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "session not found")
		return
	}
	c.JSON(http.StatusOK, session)
}

func (h *QuizHandler) ListSubjects(c *gin.Context) {
	subjects, err := h.DB.ListSubjects(c.Request.Context())
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to list subjects")
		return
	}
	c.JSON(http.StatusOK, subjects)
}
