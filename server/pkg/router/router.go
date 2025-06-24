package router

import (
	"Server/pkg/handler"
	"Server/pkg/manager"
	"Server/pkg/middleware"
	"Server/pkg/websocket"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"time"
)

func SetupRouter(rmqClient *manager.RabbitMQClient, db *gorm.DB, mgr *manager.WorkerManager, wsHub *websocket.Hub) *gin.Engine {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	apiV1 := router.Group("/api/v1")
	{
		tasks := apiV1.Group("/tasks")
		{
			tasks.POST("", handler.CreateTaskHandler(db, rmqClient))
			tasks.GET("", handler.GetTasksHandler(db))
			tasks.GET("/:id", handler.GetTaskByIDHandler(db))
			tasks.DELETE("/:id", handler.DeleteTaskHandler(db))
			tasks.POST("/accept", middleware.WorkerAuthMiddleware(db, mgr), handler.AcceptTaskHandler(db))
			tasks.PATCH("/:id", handler.UpdateTaskStatusHandler(db))
			tasks.POST("/:id/artifact", middleware.WorkerAuthMiddleware(db, mgr), handler.UploadTaskArtifactHandler(db))
		}

		workers := apiV1.Group("/workers")
		{
			workers.POST("/register", handler.RegisterWorkerHandler(db, mgr))
			workers.POST("/unregister", handler.UnregisterWorkerHandler(db, mgr))
			workers.POST("/ping", middleware.WorkerAuthMiddleware(db, mgr), handler.PingHandler(mgr))
		}

		logs := apiV1.Group("/logs")
		{
			logs.GET("/ws", func(c *gin.Context) {
				websocket.ServeWs(wsHub, c.Writer, c.Request)
			})
		}

		apiV1.GET("/artifacts/:id", handler.DownloadTaskArtifactHandler(db))
	}

	return router
}
