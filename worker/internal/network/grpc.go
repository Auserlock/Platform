package network

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"worker/internal/config"
	pb "worker/internal/proto"

	log "github.com/sirupsen/logrus"
)

type TaskStatus string

const (
	StatusPending TaskStatus = "pending"
	StatusRunning TaskStatus = "running"
	StatusSuccess TaskStatus = "success"
	StatusFailed  TaskStatus = "failed"
)

var (
	clientID   = flag.String("id", "worker1", "The unique ID for this client")
	ServerAddr = flag.String("server", fmt.Sprintf("%s:50051", config.GlobalWorker.IPAddress), "The server address in the format of host:port")
	Command    = flag.String("Command", "", "An initial Command to execute and stream its output. If empty, client only polls for commands.")
)

// uploadArtifact 上传任务产物到服务器
func uploadArtifact(ctx context.Context, httpClient *HttpClient, taskID string, localFilePath string) error {
	log.WithFields(log.Fields{
		"task_id":  taskID,
		"filepath": localFilePath,
	}).Info("preparing to upload artifact")

	// 确保文件存在
	if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", localFilePath)
	}

	// 准备上传内容
	fileFields := map[string]string{
		"artifact": localFilePath,
	}

	uploadPath := fmt.Sprintf("/api/v1/tasks/%s/artifact", taskID)

	// 执行上传
	resp, err := httpClient.PostMultipartStream(uploadPath, nil, fileFields)
	if err != nil {
		return fmt.Errorf("failed to upload artifact: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("failed to upload artifact, server response: %d - %s", resp.StatusCode, resp.String())
	}

	log.WithField("filepath", localFilePath).Info("artifact uploaded successfully")
	return nil
}

// ExecuteAndStreamLogs 执行命令并流式传输日志
func ExecuteAndStreamLogs(ctx context.Context, client pb.LogStreamServiceClient, cmdStr string, httpClient *HttpClient) error {
	log.WithField("command", cmdStr).Info("starting to execute command")

	// 解析命令
	cmdParts := strings.Fields(cmdStr)
	if len(cmdParts) == 0 {
		return fmt.Errorf("command cannot be empty")
	}

	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	cmd.Dir = "../build-vmcore"

	// 创建管道
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.WithError(err).Error("failed to create stdout pipe")
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.WithError(err).Error("failed to create stderr pipe")
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// 创建日志流
	stream, err := client.UploadLogs(ctx)
	if err != nil {
		log.WithError(err).Error("failed to create log stream")
		return fmt.Errorf("failed to create log stream: %w", err)
	}
	defer func() {
		if closeErr := stream.CloseSend(); closeErr != nil {
			log.WithError(closeErr).Warn("failed to close log stream")
		}
	}()

	// 启动命令
	if err := cmd.Start(); err != nil {
		log.WithError(err).Error("failed to start command")
		return fmt.Errorf("failed to start command: %w", err)
	}

	// 并发处理输出流
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := streamPipe(stream, stdoutPipe, "stdout", ctx); err != nil {
			errChan <- fmt.Errorf("failed to process stdout: %w", err)
		}
	}()

	go func() {
		defer wg.Done()
		if err := streamPipe(stream, stderrPipe, "stderr", ctx); err != nil {
			errChan <- fmt.Errorf("failed to process stderr: %w", err)
		}
	}()

	// 等待所有goroutine完成
	wg.Wait()
	close(errChan)

	// 检查流处理错误
	for streamErr := range errChan {
		log.WithError(streamErr).Warn("stream processing error occurred")
	}

	// 接收响应
	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.WithError(err).Error("failed to receive log stream response")
	} else {
		log.WithFields(log.Fields{
			"success": resp.Success,
			"message": resp.Message,
		}).Info("log stream response received")
	}

	// 等待命令完成并处理结果
	cmdErr := cmd.Wait()
	if err := reportTaskResult(ctx, httpClient, cmdErr); err != nil {
		log.WithError(err).Error("failed to report task result")
	}

	log.WithField("command", cmdStr).Info("command execution completed")
	return cmdErr
}

// reportTaskResult 报告任务执行结果
func reportTaskResult(ctx context.Context, httpClient *HttpClient, cmdErr error) error {
	taskID, ok := ctx.Value("taskID").(string)
	if !ok {
		return fmt.Errorf("cannot get taskID from context")
	}

	var payload map[string]interface{}
	if cmdErr != nil {
		var resultMessage string
		var exitErr *exec.ExitError
		if errors.As(cmdErr, &exitErr) {
			resultMessage = fmt.Sprintf("command execution failed with exit code: %d", exitErr.ExitCode())
		} else {
			resultMessage = fmt.Sprintf("command startup failed: %v", cmdErr)
		}
		payload = map[string]interface{}{
			"status": StatusFailed,
			"result": resultMessage,
		}
	} else {
		payload = map[string]interface{}{
			"status": StatusSuccess,
			"result": "task executed successfully",
		}
		if err := uploadTaskArtifact(ctx, httpClient); err != nil {
			log.WithError(err).Error("failed to upload artifact")
			payload = map[string]interface{}{
				"status": StatusFailed,
				"result": "failed to upload artifact",
			}
		}
	}

	taskUpdatePath := fmt.Sprintf("/api/v1/tasks/%s", taskID)
	reportCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	response, err := httpClient.PatchWithContext(reportCtx, taskUpdatePath, payload)
	if err != nil {
		slog.Error("failed to report status to server",
			slog.String("task_id", taskID),
			slog.String("error", err.Error()))
		return err
	}

	if !response.IsSuccess() {
		slog.Error("server failed to update status",
			slog.String("task_id", taskID),
			slog.Int("status_code", response.StatusCode),
			slog.String("response", response.String()))
		return fmt.Errorf("server failed to update status: %d", response.StatusCode)
	}

	slog.Info("server status updated successfully", slog.String("task_id", taskID))
	return nil
}

// uploadTaskArtifact 上传任务产物
func uploadTaskArtifact(ctx context.Context, httpClient *HttpClient) error {
	taskID, ok := ctx.Value("taskID").(string)
	if !ok {
		return fmt.Errorf("cannot get taskID from context")
	}

	taskCommit, ok := ctx.Value("taskCommit").(string)
	if !ok {
		return fmt.Errorf("cannot get taskCommit from context")
	}

	rootPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	artifactPath := filepath.Join(rootPath,
		fmt.Sprintf("../build-vmcore/build/%s/linux-%s.tar.zst", taskCommit, taskCommit))

	return uploadArtifact(ctx, httpClient, taskID, artifactPath)
}

// streamPipe 处理管道流并发送到日志服务
func streamPipe(stream pb.LogStreamService_UploadLogsClient, reader io.Reader, streamType string, ctx context.Context) error {
	taskID, ok := ctx.Value("taskID").(string)
	workID, ok := ctx.Value("workerID").(string)
	if !ok {
		return fmt.Errorf("cannot get taskID from context")
	}

	scanner := bufio.NewScanner(reader)
	// 设置更大的缓冲区以处理长行
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		fmt.Printf("%s\n", scanner.Text())

		// 根据proto文件，LogMessage只有client_id, task_id, timestamp, message字段
		msg := &pb.LogMessage{
			ClientId:  workID,
			TaskId:    taskID,
			Message:   scanner.Text(),
			Timestamp: time.Now().Format(time.RFC3339Nano),
		}

		if err := stream.Send(msg); err != nil {
			log.WithError(err).Error("failed to send log message")
			return fmt.Errorf("failed to send log message: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.WithFields(log.Fields{
			"stream_type": streamType,
			"error":       err,
		}).Error("error reading from pipe")
		return fmt.Errorf("error reading from pipe (%s): %w", streamType, err)
	}

	return nil
}

// pollForCommands 轮询服务器获取命令
func pollForCommands(ctx context.Context, client pb.CommandServiceClient) {
	log.Info("starting to poll for commands every 10 seconds")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("stopping command polling")
			return
		case <-ticker.C:
			if err := pollSingleCommand(ctx, client); err != nil {
				log.WithError(err).Error("failed to poll command")
			}
		}
	}
}

// pollSingleCommand 执行单次命令轮询
func pollSingleCommand(ctx context.Context, client pb.CommandServiceClient) error {
	ping := &pb.PingCommand{
		ClientId: *clientID,
		Time:     time.Now().Format(time.RFC3339Nano),
	}

	log.Debug("sending ping request to server")

	// 添加超时控制
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd, err := client.GetCommand(pingCtx, ping)
	if err != nil {
		return fmt.Errorf("failed to get command: %w", err)
	}

	if cmd.NoCommandAvailable {
		log.Debug("no command available from server")
		return nil
	}

	handleCommand(ctx, cmd)
	return nil
}

// handleCommand 处理接收到的命令
func handleCommand(ctx context.Context, cmd *pb.Command) {
	log.WithFields(log.Fields{
		"command_id":    cmd.CommandId,
		"type":          cmd.Type.String(),
		"target_client": cmd.TargetClient,
		"payload":       cmd.Payload,
	}).Info("received command")

	switch cmd.Type {
	case pb.CommandType_PONG:
		log.Info("received PONG command")

	case pb.CommandType_EXECUTE_SHELL:
		handleShellCommand(ctx, cmd)

	case pb.CommandType_OPEN_SSH, pb.CommandType_OPEN_QEMU_MONITOR:
		log.WithField("type", cmd.Type.String()).Warn("received command type, but interactive sessions are not yet implemented")
		// TODO: 在这里可以添加启动 TransportService 双向流的逻辑

	case pb.CommandType_RESTART_SERVICE:
		log.WithField("type", cmd.Type.String()).Info("received restart service command")
		// TODO: 实现服务重启逻辑

	case pb.CommandType_CUSTOM:
		log.WithField("type", cmd.Type.String()).Info("received custom command")
		// TODO: 实现自定义命令处理逻辑

	default:
		log.WithField("type", cmd.Type.String()).Warn("received unknown or unsupported command type")
	}
}

// handleShellCommand 处理shell命令执行
func handleShellCommand(ctx context.Context, cmd *pb.Command) {
	log.WithField("payload", cmd.Payload).Info("executing shell command")

	if strings.TrimSpace(cmd.Payload) == "" {
		log.Error("shell command payload is empty")
		return
	}

	execCtx, cancel := context.WithTimeout(ctx, 5*time.Minute) // 增加超时时间
	defer cancel()

	cmdParts := strings.Fields(cmd.Payload)
	shellCmd := exec.CommandContext(execCtx, cmdParts[0], cmdParts[1:]...)

	output, err := shellCmd.CombinedOutput()
	if err != nil {
		log.WithFields(log.Fields{
			"command": cmd.Payload,
			"error":   err,
			"output":  string(output),
		}).Error("shell command execution failed")
	} else {
		log.WithFields(log.Fields{
			"command": cmd.Payload,
			"output":  string(output),
		}).Info("shell command executed successfully")
	}
}
