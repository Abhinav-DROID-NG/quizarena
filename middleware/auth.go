package middleware

import (
	"net/http"
	"strings"

	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const UserIDKey = "userID"

type tokenParser interface {
	ParseToken(raw string) (int64, map[string]any, error)
}

func JWTAuth(tokenManager tokenParser) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			utils.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token")
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if isTokenBlacklisted(c, token) {
			utils.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "token has been invalidated")
			return
		}
		userID, _, err := tokenManager.ParseToken(token)
		if err != nil {
			utils.RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
			return
		}
		c.Set(UserIDKey, userID)
		c.Next()
	}
}

func isTokenBlacklisted(c *gin.Context, token string) bool {
	redisClient, ok := c.Get("redisClient")
	if !ok {
		return false
	}
	client, ok := redisClient.(*redis.Client)
	if !ok || client == nil {
		return false
	}
	_, err := client.Get(c.Request.Context(), "blacklist:"+token).Result()
	return err == nil
}
