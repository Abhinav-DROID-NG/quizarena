package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/Abhinav-DROID-NG/quizarena/middleware"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"
)

const googleVerifyTimeout = 3 * time.Second

type GoogleVerifier interface {
	Verify(ctx context.Context, token string, audience string) (*idtoken.Payload, error)
}

type googleVerifier struct{}

func (g googleVerifier) Verify(ctx context.Context, token string, audience string) (*idtoken.Payload, error) {
	return idtoken.Validate(ctx, token, audience)
}

type tokenManager interface {
	GenerateToken(userID int64, email string) (string, error)
	ParseToken(raw string) (int64, map[string]any, error)
}

type AuthHandler struct {
	DB             *database.Client
	TokenManager   tokenManager
	GoogleClientID string
	Verifier       GoogleVerifier
}

type GoogleAuthRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
}

func NewAuthHandler(db *database.Client, tm tokenManager, clientID string) *AuthHandler {
	return &AuthHandler{DB: db, TokenManager: tm, GoogleClientID: clientID, Verifier: googleVerifier{}}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}
	if err := utils.ValidateEmail(req.Email); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "invalid email")
		return
	}
	if err := utils.ValidateName(req.Name); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "invalid name")
		return
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to process password")
		return
	}

	user, err := h.DB.CreateUser(c.Request.Context(), req.Email, hash, req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
			utils.RespondError(c, http.StatusConflict, "USER_EXISTS", "email already registered")
			return
		}
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to create user")
		return
	}

	token, err := h.TokenManager.GenerateToken(user.ID, user.Email)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "TOKEN_ERROR", "failed to generate token")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"token": token, "user": user})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}
	if err := utils.ValidateEmail(req.Email); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "INVALID_INPUT", "invalid email")
		return
	}

	user, hash, err := h.DB.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
		return
	}

	if hash == "" {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "this account uses Google login")
		return
	}

	if err := utils.ComparePassword(hash, req.Password); err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
		return
	}

	token, err := h.TokenManager.GenerateToken(user.ID, user.Email)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "TOKEN_ERROR", "failed to generate token")
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (h *AuthHandler) GoogleAuth(c *gin.Context) {
	var req GoogleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "id_token is required")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), googleVerifyTimeout)
	defer cancel()
	payload, err := h.Verifier.Verify(ctx, req.IDToken, h.GoogleClientID)
	if err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_GOOGLE_TOKEN", "invalid google token")
		return
	}
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)
	sub := payload.Subject
	if strings.TrimSpace(sub) == "" {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_GOOGLE_TOKEN", "invalid google subject")
		return
	}
	if err := utils.ValidateEmail(email); err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_GOOGLE_TOKEN", "invalid google email")
		return
	}
	if strings.TrimSpace(name) == "" {
		name = "QuizArena User"
	}
	if err := utils.ValidateName(name); err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_GOOGLE_TOKEN", "invalid google profile")
		return
	}

	user, err := h.DB.UpsertOAuthUser(c.Request.Context(), sub, email, name, picture)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "DB_ERROR", "failed to upsert user")
		return
	}
	token, err := h.TokenManager.GenerateToken(user.ID, user.Email)
	if err != nil {
		utils.RespondError(c, http.StatusInternalServerError, "TOKEN_ERROR", "failed to create token")
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "missing bearer token")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	uidRaw, ok := c.Get(middleware.UserIDKey)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context")
		return
	}
	uid, ok := uidRaw.(int64)
	if !ok {
		utils.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user context")
		return
	}
	user, err := h.DB.GetUserByID(c.Request.Context(), uid)
	if err != nil {
		utils.RespondError(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}
	c.JSON(http.StatusOK, user)
}
