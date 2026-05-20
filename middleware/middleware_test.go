package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestJWTAuthValidAndInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tm := utils.NewTokenManager("secret", time.Hour)
	valid, err := tm.GenerateToken(42, "user@example.com")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	r := gin.New()
	r.GET("/secure", JWTAuth(tm), func(c *gin.Context) { c.Status(http.StatusOK) })

	badReq := httptest.NewRequest(http.MethodGet, "/secure", nil)
	badReq.Header.Set("Authorization", "Bearer invalid")
	badResp := httptest.NewRecorder()
	r.ServeHTTP(badResp, badReq)
	if badResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token got %d", badResp.Code)
	}

	goodReq := httptest.NewRequest(http.MethodGet, "/secure", nil)
	goodReq.Header.Set("Authorization", "Bearer "+valid)
	goodResp := httptest.NewRecorder()
	r.ServeHTTP(goodResp, goodReq)
	if goodResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid token got %d", goodResp.Code)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mu.Lock()
	limiters = map[string]*ipLimiter{}
	mu.Unlock()

	r := gin.New()
	r.Use(RateLimitMiddleware(rate.Limit(1), 1))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "1.2.3.4:1234"
	resp1 := httptest.NewRecorder()
	r.ServeHTTP(resp1, req1)
	if resp1.Code != http.StatusOK {
		t.Fatalf("first request should pass, got %d", resp1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "1.2.3.4:1234"
	resp2 := httptest.NewRecorder()
	r.ServeHTTP(resp2, req2)
	if resp2.Code != http.StatusTooManyRequests {
		t.Fatalf("second immediate request should be rate limited, got %d", resp2.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS([]string{"https://frontend.example.com"}))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://frontend.example.com")
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)

	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "https://frontend.example.com" {
		t.Fatalf("unexpected allow-origin header: %q", got)
	}

	optReq := httptest.NewRequest(http.MethodOptions, "/", nil)
	optReq.Header.Set("Origin", "https://frontend.example.com")
	optResp := httptest.NewRecorder()
	r.ServeHTTP(optResp, optReq)
	if optResp.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for preflight got %d", optResp.Code)
	}
}
