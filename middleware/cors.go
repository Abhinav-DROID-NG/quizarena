package middleware

import (
"net/http"
"strings"

"github.com/gin-gonic/gin"
)

func CORS(allowedOrigins []string) gin.HandlerFunc {
allowAll := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"
originSet := map[string]struct{}{}
for _, origin := range allowedOrigins {
originSet[strings.TrimSpace(origin)] = struct{}{}
}

return func(c *gin.Context) {
origin := c.GetHeader("Origin")
if origin != "" {
if allowAll {
c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
} else if _, ok := originSet[origin]; ok {
c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
c.Writer.Header().Set("Vary", "Origin")
}
}
c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

if c.Request.Method == http.MethodOptions {
c.AbortWithStatus(http.StatusNoContent)
return
}
c.Next()
}
}
