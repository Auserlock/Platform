package handler

import (
	"Server/pkg/manager"
	"Server/pkg/middleware"
	"Server/pkg/model"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RegisterWorkerRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
	APIKey   string `json:"api_key,omitempty"`
	Hostname string `json:"hostname"`
}

func RegisterWorkerHandler(db *gorm.DB, mgr *manager.WorkerManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterWorkerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}

		var worker model.Worker
		result := db.Where("worker_id = ?", req.WorkerID).First(&worker)

		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			newApiKey := middleware.GenerateSecureAPIKey()

			hashedApiKey, err := middleware.HashAPIKey(newApiKey)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash API key"})
				return
			}

			newWorker := model.Worker{
				WorkerID: req.WorkerID,
				APIKey:   hashedApiKey,
				Hostname: req.Hostname,
				Status:   "online",
				LastSeen: func() *time.Time { t := time.Now(); return &t }(),
			}

			if err := db.Create(&newWorker).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new worker record"})
				return
			}
			mgr.Register(newWorker.WorkerID, newWorker.Hostname)
			c.JSON(http.StatusCreated, gin.H{
				"status":    "created",
				"message":   "New worker created. Please save your API key securely.",
				"worker_id": newWorker.WorkerID,
				"api_key":   newApiKey,
			})
			return
		}
		if result.Error == nil {
			if req.APIKey == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is required for existing worker"})
				return
			}
			slog.Info(req.APIKey, worker.APIKey)
			if !middleware.CheckAPIKeyHash(req.APIKey, worker.APIKey) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Invalid API key"})
				return
			}

			now := time.Now()
			updates := map[string]interface{}{
				"status":    "online",
				"last_seen": &now,
				"hostname":  req.Hostname,
			}
			if err := db.Model(&worker).Updates(updates).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update worker status"})
				return
			}
			mgr.Register(worker.WorkerID, req.Hostname)

			c.JSON(http.StatusOK, gin.H{
				"status":    "success",
				"message":   "Existing worker is now online.",
				"worker_id": worker.WorkerID,
				"api_key":   req.APIKey,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + result.Error.Error()})
	}
}

func PingHandler(mgr *manager.WorkerManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get("worker")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Worker context not found"})
			return
		}

		worker := val.(*model.Worker)

		if mgr.Ping(worker.WorkerID) {
			c.JSON(http.StatusOK, gin.H{"status": "pong"})
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "Failed to ping. Worker may have just gone offline."})
		}
	}
}

func UnregisterWorkerHandler(db *gorm.DB, mgr *manager.WorkerManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterWorkerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}

		var worker model.Worker
		result := db.Where("worker_id = ?", req.WorkerID).First(&worker)

		if result.Error == nil {
			if req.APIKey == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "API key is required for existing worker"})
				return
			}

			if !middleware.CheckAPIKeyHash(req.APIKey, worker.APIKey) {
				c.JSON(http.StatusForbidden, gin.H{"error": "Invalid API key"})
				return
			}

			now := time.Now()
			updates := map[string]interface{}{
				"status":    "offline",
				"last_seen": &now,
				"hostname":  req.Hostname,
			}
			if err := db.Model(&worker).Updates(updates).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update worker status"})
				return
			}
			mgr.Unregister(worker.WorkerID)

			c.JSON(http.StatusOK, gin.H{
				"status":    "success",
				"message":   "Existing worker is now offline.",
				"worker_id": worker.WorkerID,
				"api_key":   worker.APIKey,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + result.Error.Error()})
	}
}
