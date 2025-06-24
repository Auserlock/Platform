package handler

import (
	"Server/pkg/manager"
	"Server/pkg/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io"
	"log"
	"log/slog"
	"net/http"
	"time"
)

func CreateTaskHandler(db *gorm.DB, rmqClient *manager.RabbitMQClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskTypeStr := c.PostForm("task_type")
		taskType := model.TaskType(taskTypeStr)

		var task *model.Task
		var err error

		switch taskType {
		case model.TaskTypeKernelBuild:
			slog.Info("handling 'kernel-build' task...")
			var report model.CrashReport

			file, err := c.FormFile("report")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get file from form: " + err.Error()})
				return
			}

			openedFile, err := file.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open uploaded file: " + err.Error()})
				return
			}
			defer func() {
				if err := openedFile.Close(); err != nil {
					log.Println(err)
				}
			}()

			if err = json.NewDecoder(openedFile).Decode(&report); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse JSON content from file: " + err.Error()})
				return
			}

			task = model.CreateTask(taskType, report)
			slog.Info("new 'kernel-build' task created with ID: %s", task.ID)

		case model.TaskTypePatchApply:
			slog.Info("handling 'patch-apply' task...")

			existingTaskUUID := c.PostForm("uuid")
			if existingTaskUUID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'uuid' for patch-apply task"})
				return
			}

			var existingTask model.Task

			if err = db.First(&existingTask, "id = ?", existingTaskUUID).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Task with UUID '%s' not found: %v", existingTaskUUID, err)})
				return
			}
			slog.Info("Found existing task %s to apply patch.", existingTask.ID)

			patchFile, err := c.FormFile("patch")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get patch file from form: " + err.Error()})
				return
			}

			openedPatchFile, err := patchFile.Open()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open patch file: " + err.Error()})
				return
			}
			defer func() {
				if err := openedPatchFile.Close(); err != nil {
					log.Println(err)
				}
			}()

			patchBytes, err := io.ReadAll(openedPatchFile)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read patch file content: " + err.Error()})
				return
			}
			patchContent := string(patchBytes)

			newReport := existingTask.Payload
			newReport.Patch = patchContent

			task = model.CreateTask(taskType, newReport)
			slog.Info("new 'patch-apply' task created with ID: %s based on old task %s", task.ID, existingTask.ID)

		default:
			errorMsg := fmt.Sprintf("Invalid task type: '%s'. Must be one of '%s' or '%s'",
				taskTypeStr, model.TaskTypeKernelBuild, model.TaskTypePatchApply)
			c.JSON(http.StatusBadRequest, gin.H{"error": errorMsg})
			return
		}

		if err = db.Create(&task).Error; err != nil {
			slog.Info("failed to save task to DB: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save task to database"})
			return
		}
		slog.Info("task %s successfully saved to database.", task.ID)

		taskJSON, err := json.Marshal(task)
		if err != nil {
			slog.Info("error marshalling task to JSON: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize task object"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if err = rmqClient.Publish(ctx, string(taskJSON)); err != nil {
			slog.Info("failed to publish a message: %s", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit task to queue"})
			return
		}

		slog.Info("successfully published task %s to queue", task.ID)

		c.JSON(http.StatusAccepted, task)
	}
}

func GetTasksHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tasks []model.Task
		if err := db.Order("created_at desc").Find(&tasks).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
			return
		}
		c.JSON(http.StatusOK, tasks)
	}
}

func GetTaskByIDHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("id")
		var task model.Task

		if err := db.First(&task, "id = ?", taskID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			}
			return
		}
		c.JSON(http.StatusOK, task)
	}
}

func DeleteTaskHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")

		taskID, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
			return
		}

		result := db.Delete(&model.Task{}, "id = ?", taskID)

		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
			return
		}

		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func AcceptTaskHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		type RequestBody struct {
			ID       string `form:"id"`
			WorkerID string `form:"worker_id"`
		}

		var req RequestBody
		if err := c.ShouldBind(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form"})
			return
		}

		taskIDStr := req.ID
		taskID, err := uuid.Parse(taskIDStr)
		if err != nil {
			slog.Error("invalid task ID format", "id", taskIDStr, "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
			return
		}

		workerID := req.WorkerID

		var updatedTask model.Task

		err = db.Transaction(func(tx *gorm.DB) error {
			var task model.Task

			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, "id = ?", taskID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("task not found")
				}
				return err
			}

			if task.WorkerID != "" {
				return fmt.Errorf("task already accepted by worker %s", task.WorkerID)
			}

			task.WorkerID = workerID
			task.Status = model.StatusRunning
			now := time.Now()
			task.StartedAt = &now

			if err := tx.Save(&task).Error; err != nil {
				return err
			}

			updatedTask = task
			return nil
		})

		if err != nil {
			slog.Error("failed to accept task", "task_id", taskID, "worker_id", workerID, "error", err)

			if err.Error() == "task not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			} else if len(err.Error()) > 20 && err.Error()[:20] == "task already accepted" {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task in database"})
			}
			return
		}

		slog.Info("task accepted successfully", "task_id", updatedTask.ID, "worker_id", updatedTask.WorkerID)
		c.JSON(http.StatusOK, updatedTask)
	}
}

// UpdateTaskStatusHandler 用于更新一个任务的状态.
// 通常由 Worker 在任务完成后调用.
func UpdateTaskStatusHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 定义请求体结构
		type RequestBody struct {
			Status model.TaskStatus `json:"status" binding:"required"`
			Result string           `json:"result"`
		}

		// 2. 从 URL 中获取任务 ID
		idStr := c.Param("id")
		taskID, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID format"})
			return
		}

		// 3. 绑定并验证请求体
		var reqBody RequestBody
		if err := c.ShouldBindJSON(&reqBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
			return
		}

		// 4. 校验状态是否为有效的终结状态
		if reqBody.Status != model.StatusSuccess && reqBody.Status != model.StatusFailed {
			errorMsg := fmt.Sprintf("Invalid status update: '%s'. Status can only be updated to '%s' or '%s'",
				reqBody.Status, model.StatusSuccess, model.StatusFailed)
			c.JSON(http.StatusBadRequest, gin.H{"error": errorMsg})
			return
		}

		var updatedTask model.Task

		// 5. 使用事务来更新数据库，确保原子性
		err = db.Transaction(func(tx *gorm.DB) error {
			var task model.Task

			// 查找任务
			if err := tx.First(&task, "id = ?", taskID).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return fmt.Errorf("task not found")
				}
				return err // 其他数据库错误
			}

			// 准备要更新的字段
			updateFields := map[string]interface{}{
				"status":      reqBody.Status,
				"result":      reqBody.Result,
				"finished_at": time.Now().UTC(),
			}

			// 执行更新操作
			if err := tx.Model(&task).Updates(updateFields).Error; err != nil {
				return err
			}

			updatedTask = task
			// 更新本地 updatedTask 变量以反映更改
			updatedTask.Status = reqBody.Status
			updatedTask.Result = reqBody.Result
			now := time.Now().UTC()
			updatedTask.FinishedAt = &now

			return nil // 提交事务
		})

		// 6. 处理事务执行结果
		if err != nil {
			slog.Error("failed to update task status", "task_id", taskID, "error", err)
			if err.Error() == "task not found" {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task in database"})
			}
			return
		}

		slog.Info("task status updated successfully", "task_id", updatedTask.ID, "new_status", updatedTask.Status)
		c.JSON(http.StatusOK, updatedTask)
	}
}
