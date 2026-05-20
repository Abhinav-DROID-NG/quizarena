package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/Abhinav-DROID-NG/quizarena/database"
	"github.com/Abhinav-DROID-NG/quizarena/middleware"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"
)

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

func NewAuthHandler(db *database.Client, tm tokenManager, clientID string) *AuthHandler {
	return &AuthHandler{DB: db, TokenManager: tm, GoogleClientID: clientID, Verifier: googleVerifier{}}
}

func (h *AuthHandler) GoogleAuth(c *gin.Context) {
	var req GoogleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondError(c, http.StatusBadRequest, "BAD_REQUEST", "id_token is required")
		return
	}
	payload, err := h.Verifier.Verify(c.Request.Context(), req.IDToken, h.GoogleClientID)
	if err != nil {
		utils.RespondError(c, http.StatusUnauthorized, "INVALID_GOOGLE_TOKEN", "invalid google token")
		return
	}
	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)
	sub := payload.Subject

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
