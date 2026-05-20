package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Abhinav-DROID-NG/quizarena/middleware"
	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
)

func TestJWTMiddlewareRejectsMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tm := utils.NewTokenManager("test-secret", time.Hour)
	r := gin.New()
	r.GET("/secure", middleware.JWTAuth(tm), func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", resp.Code)
	}
}
