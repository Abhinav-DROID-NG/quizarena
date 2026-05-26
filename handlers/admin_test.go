package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Abhinav-DROID-NG/quizarena/middleware"
	"github.com/gin-gonic/gin"
)

func TestAdminListQuestionsRequiresAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAdminHandler(nil, []string{"admin@example.com"})
	r := gin.New()
	r.GET("/admin/questions", func(c *gin.Context) {
		c.Set(middleware.UserClaimsKey, map[string]any{"email": "user@example.com"})
		h.ListQuestions(c)
	})
	req := httptest.NewRequest(http.MethodGet, "/admin/questions", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 got %d", resp.Code)
	}
}
