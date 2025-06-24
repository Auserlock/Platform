package middleware

import (
	"net/http"

	"Server/pkg/manager"
	"Server/pkg/model"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func WorkerAuthMiddleware(db *gorm.DB, mgr *manager.WorkerManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {WorkerID}:{APIKeyID}.{SecretAPIKey}"})
			return
		}
		tokenString := parts[1]

		idAndKeyParts := strings.SplitN(tokenString, ":", 2)
		if len(idAndKeyParts) != 2 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token format must be {WorkerID}:{APIKeyID}"})
			return
		}
		workerID := idAndKeyParts[0]
		apiKey := idAndKeyParts[1]

		var authenticatedWorker *model.Worker
		db.First(&authenticatedWorker, "worker_id = ?", workerID)

		if authenticatedWorker == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Invalid API Key"})
			return
		}

		if !CheckAPIKeyHash(apiKey, authenticatedWorker.APIKey) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API Key"})
		}

		if !mgr.IsOnline(authenticatedWorker.WorkerID) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Worker is considered offline. Please re-register."})
			return
		}

		c.Set("worker", authenticatedWorker)
		c.Next()
	}
}
