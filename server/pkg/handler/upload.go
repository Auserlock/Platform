package handler

import (
	"Server/pkg/model"
	"fmt"
	"io" // 导入 io 包以使用 io.Copy
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const artifactDir = "./artifacts"

// UploadTaskArtifactStreamHandler 处理 Worker 上传的产物文件（流式）
func UploadTaskArtifactHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("id")

		// 1. 查找任务是否存在 (逻辑不变)
		var task model.Task
		if err := db.First(&task, "id = ?", taskID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
			return
		}

		// --- 流式处理核心改动 ---
		// 2. 直接从请求中获取 multipart reader，而不是一次性解析整个表单
		reader, err := c.Request.MultipartReader()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "获取 multipart reader 失败: " + err.Error()})
			return
		}

		// 3. 遍历 multipart 的各个部分，直到找到名为 "artifact" 的文件部分
		// 这避免了在找到文件前解析不必要的数据
		part, err := reader.NextPart()
		for err == nil && part.FormName() != "artifact" {
			part, err = reader.NextPart()
		}

		// 检查是否成功找到了文件部分
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "找不到名为 'artifact' 的文件部分: " + err.Error()})
			return
		}

		fileName := part.FileName()
		if fileName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件名为空"})
			return
		}

		// 4. 创建用于存储该任务产物的目录 (逻辑不变)
		taskArtifactDir := filepath.Join(artifactDir, taskID)
		if err := os.MkdirAll(taskArtifactDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建产物目录"})
			return
		}

		// 5. 创建目标文件，准备写入数据
		artifactPath := filepath.Join(taskArtifactDir, fileName)
		dst, err := os.Create(artifactPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建目标文件: " + err.Error()})
			return
		}
		defer dst.Close()

		// 6. 【核心】使用 io.Copy 将文件内容从请求体直接流式传输到目标文件
		// 数据以块(chunk)的形式被读取和写入，内存占用非常小。
		if _, err := io.Copy(dst, part); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存产物文件失败: " + err.Error()})
			return
		}
		// --- 流式处理核心改动结束 ---

		// 7. 更新数据库，记录产物路径和文件名 (逻辑不变)
		updates := map[string]interface{}{
			"artifact_path": artifactPath,
			"artifact_name": fileName,
		}
		if err := db.Model(&task).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新任务数据库失败"})
			return
		}

		slog.Info("产物流式上传成功", "task_id", taskID, "path", artifactPath)
		c.JSON(http.StatusOK, gin.H{"message": "产物上传成功"})
	}
}

// DownloadTaskArtifactHandler DownloadTaskArtifactStreamHandler 处理用户下载产物文件的请求（流式）
func DownloadTaskArtifactHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		taskID := c.Param("id")

		// 1. 查找任务以获取文件信息 (逻辑不变)
		var task model.Task
		if err := db.First(&task, "id = ?", taskID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
			return
		}

		// 2. 检查文件路径是否存在 (逻辑不变)
		if task.ArtifactPath == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "该任务没有关联的产物文件"})
			return
		}

		// --- 流式处理核心改动 ---
		// 3. 打开文件准备读取
		file, err := os.Open(task.ArtifactPath)
		if err != nil {
			if os.IsNotExist(err) {
				c.JSON(http.StatusNotFound, gin.H{"error": "产物文件在服务器上不存在"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "无法打开产物文件"})
			}
			return
		}
		defer file.Close()

		// 获取文件信息，用于设置 Content-Length 头
		fileInfo, err := file.Stat()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取文件信息"})
			return
		}

		// 4. 设置响应头 (逻辑不变，但增加了 Content-Length)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", task.ArtifactName))
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size())) // 明确告诉浏览器文件大小

		// 5. 【核心】使用 io.Copy 将文件内容直接流式传输到 HTTP 响应写入器
		// 这会从磁盘读取文件块并直接写入网络套接字，而不会将整个文件加载到内存中。
		if _, err := io.Copy(c.Writer, file); err != nil {
			// 当传输中途发生错误时，可能无法再向客户端发送JSON错误（因为头已发送），
			// 所以在这里记录日志是最好的方式。
			slog.Error("文件流式传输期间发生错误", "task_id", taskID, "error", err)
		}
		// --- 流式处理核心改动结束 ---
	}
}
