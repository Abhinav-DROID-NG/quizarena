package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/Abhinav-DROID-NG/quizarena/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	mu       sync.Mutex
	limiters = map[string]*ipLimiter{}
)

func RateLimitMiddleware(rps rate.Limit, burst int) gin.HandlerFunc {
	return rateLimitMiddleware(rps, burst, func(c *gin.Context) string {
		return "global:" + c.ClientIP()
	})
}

func AuthRateLimitMiddleware(rps rate.Limit, burst int) gin.HandlerFunc {
	return rateLimitMiddleware(rps, burst, func(c *gin.Context) string {
		return "auth:" + c.FullPath() + ":" + c.ClientIP()
	})
}

func rateLimitMiddleware(rps rate.Limit, burst int, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	go cleanupLimiters()
	return func(c *gin.Context) {
		limiter := getLimiter(keyFunc(c), rps, burst)
		if !limiter.Allow() {
			utils.RespondError(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
			return
		}
		c.Next()
	}
}

func getLimiter(key string, rps rate.Limit, burst int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()
	entry, exists := limiters[key]
	if !exists {
		entry = &ipLimiter{limiter: rate.NewLimiter(rps, burst), lastSeen: time.Now()}
		limiters[key] = entry
		return entry.limiter
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

func cleanupLimiters() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		mu.Lock()
		for ip, limiter := range limiters {
			if time.Since(limiter.lastSeen) > 10*time.Minute {
				delete(limiters, ip)
			}
		}
		mu.Unlock()
	}
}
