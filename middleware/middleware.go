package middleware

import (
	"time"

	"go-common/logger"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Logger returns a gin.HandlerFunc for logging requests
func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Create a structured log entry
		appLogger := logger.NewFromEnv("http-request")

		// Extract request ID if available
		requestID := ""
		if id, exists := param.Keys["requestId"]; exists {
			if idStr, ok := id.(string); ok {
				requestID = idStr
			}
		}

		// Log the request
		appLogger.Info("HTTP request completed", map[string]interface{}{
			"requestId":  requestID,
			"method":     param.Method,
			"url":        param.Path,
			"statusCode": param.StatusCode,
			"duration":   param.Latency.Milliseconds(),
			"userAgent":  param.Request.UserAgent(),
			"clientIP":   param.ClientIP,
		})

		return ""
	})
}

// CORS returns a gin.HandlerFunc for CORS configuration
func CORS(corsMaxAge int) gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Requested-With"}
	config.AllowCredentials = true
	config.MaxAge = time.Duration(corsMaxAge) * time.Hour

	return cors.New(config)
}
