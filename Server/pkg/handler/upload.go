package handler

import (
	"Server/pkg/model"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const artifactDir = "./artifacts"

// UploadTaskArtifactHandler 处理 Worker 上传的产物文件
func UploadTaskArtifactHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("id")

		// 1. 查找任务是否存在
		var task model.Task
		if err := db.First(&task, "id = ?", taskID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
			return
		}

		// 2. 获取上传的文件
		file, err := c.FormFile("artifact")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "获取产物文件失败: " + err.Error()})
			return
		}

		// 3. 创建用于存储该任务产物的目录
		taskArtifactDir := filepath.Join(artifactDir, taskID)
		if err := os.MkdirAll(taskArtifactDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建产物目录"})
			return
		}

		// 4. 保存文件
		artifactPath := filepath.Join(taskArtifactDir, file.Filename)
		if err := c.SaveUploadedFile(file, artifactPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存产物文件失败"})
			return
		}

		// 5. 更新数据库，记录产物路径和文件名
		updates := map[string]interface{}{
			"artifact_path": artifactPath,
			"artifact_name": file.Filename,
		}
		if err := db.Model(&task).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新任务数据库失败"})
			return
		}

		slog.Info("产物上传成功", "task_id", taskID, "path", artifactPath)
		c.JSON(http.StatusOK, gin.H{"message": "产物上传成功"})
	}
}

// DownloadTaskArtifactHandler 处理用户下载产物文件的请求
func DownloadTaskArtifactHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("id")

		// 1. 查找任务以获取文件信息
		var task model.Task
		if err := db.First(&task, "id = ?", taskID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
			return
		}

		// 2. 检查文件路径和文件是否存在
		if task.ArtifactPath == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "该任务没有关联的产物文件"})
			return
		}
		if _, err := os.Stat(task.ArtifactPath); os.IsNotExist(err) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "产物文件在服务器上丢失"})
			return
		}

		// 3. 设置响应头，让浏览器触发下载而不是预览
		// Content-Disposition: attachment 会强制下载
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", task.ArtifactName))
		// Content-Type: application/octet-stream 是一个通用的二进制文件类型
		c.Header("Content-Type", "application/octet-stream")

		// 4. 使用 Gin 高效地将文件流式传输给客户端
		c.File(task.ArtifactPath)
	}
}
