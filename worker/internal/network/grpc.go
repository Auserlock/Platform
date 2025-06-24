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
	ServerAddr = flag.String("server", "localhost:50051", "The server address in the format of host:port")
	Command    = flag.String("Command", "", "An initial Command to execute and stream its output. If empty, client only polls for commands.")
)

func uploadArtifact(ctx context.Context, httpClient *HttpClient, taskID string, localFilePath string) error {
	log.Printf("准备为任务 %s 上传产物: %s\n", taskID, localFilePath)

	// 确保文件存在
	if _, err := os.Stat(localFilePath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", localFilePath)
	}

	// 准备上传内容
	fileFields := map[string]string{
		"artifact": localFilePath,
	}

	uploadPath := fmt.Sprintf("/api/v1/tasks/%s/artifact", taskID)

	// 执行上传
	resp, err := httpClient.PostMultipartStream(uploadPath, nil, fileFields)
	if err != nil {
		return fmt.Errorf("上传产物失败: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("上传产物失败，服务器响应: %d - %s", resp.StatusCode, resp.String())
	}

	log.Printf("产物上传成功: %s\n", localFilePath)
	return nil
}

func ExecuteAndStreamLogs(ctx context.Context, client pb.LogStreamServiceClient, cmdStr string, httpClient *HttpClient) error {
	log.Infof("Starting initial Command: '%s'", cmdStr)

	cmdParts := strings.Fields(cmdStr)
	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		log.Errorf("Failed to get stdout pipe for initial Command: %v", err)
		return err
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Errorf("Failed to get stderr pipe for initial Command: %v", err)
		return err
	}

	stream, err := client.UploadLogs(ctx)
	if err != nil {
		log.Errorf("Could not open log stream: %v", err)
		return err
	}

	if err := cmd.Start(); err != nil {
		log.Errorf("Failed to start initial Command: %v", err)
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go streamPipe(stream, stdoutPipe, pb.LogLevel_INFO, &wg, ctx)
	go streamPipe(stream, stderrPipe, pb.LogLevel_ERROR, &wg, ctx)
	wg.Wait()

	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Errorf("Failed to receive closing response from LogStreamService: %v", err)
	} else {
		log.Infof("LogStreamService response: Success=%v, Message='%s'", resp.Success, resp.Message)
	}

	err = cmd.Wait()
	var payload map[string]interface{}
	if err != nil {
		// 命令失败，准备失败的 payload (省略细节)...
		var resultMessage string
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			resultMessage = fmt.Sprintf("命令执行失败, 退出码: %d.", exitErr.ExitCode())
		} else {
			resultMessage = fmt.Sprintf("命令启动失败: %v", err)
		}
		payload = map[string]interface{}{
			"status": StatusFailed,
			"result": resultMessage,
		}
	} else {
		// 命令成功，准备成功的 payload
		payload = map[string]interface{}{
			"status": StatusSuccess,
			"result": "任务成功执行完成。",
		}
	}
	taskUpdatePath := fmt.Sprintf("/api/v1/tasks/%s", ctx.Value("taskID").(string))

	// 2. 准备 context 参数 (创建一个有5秒超时的子上下文)
	reportCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// 3. 调用 PatchWithContext，传入准备好的参数
	response, patchErr := httpClient.PatchWithContext(reportCtx, taskUpdatePath, payload)

	// 4. 对调用结果进行处理
	if patchErr != nil {
		// 网络请求本身就失败了
		slog.Error("向服务端报告状态失败", "task_id", ctx.Value("taskID").(string), "error", patchErr)
		return err
	}

	if !response.IsSuccess() {
		slog.Error("服务端未能成功更新状态", "task_id", ctx.Value("taskID").(string), "status_code", response.StatusCode, "response", resp.String())
	} else {
		slog.Info("服务端状态更新成功", "task_id", ctx.Value("taskID").(string))
	}
	log.Infof("Initial Command '%s' has finished.", cmdStr)

	rootPath, _ := os.Getwd()
	err = uploadArtifact(ctx, httpClient, ctx.Value("taskID").(string), filepath.Join(rootPath, fmt.Sprintf("build/%s/linux-%s.tar.zst", ctx.Value("taskCommit").(string), ctx.Value("taskCommit").(string))))
	if err != nil {
		log.Errorf("Failed to upload artifact: %v", err)
		return err
	}

	return nil
}

func streamPipe(stream pb.LogStreamService_UploadLogsClient, reader io.Reader, level pb.LogLevel, wg *sync.WaitGroup, ctx context.Context) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		msg := &pb.LogMessage{
			ClientId:  ctx.Value("taskID").(string),
			Level:     level,
			Message:   scanner.Text(),
			Timestamp: time.Now().Format(time.RFC3339Nano),
		}
		if err := stream.Send(msg); err != nil {
			log.Errorf("Failed to send log message: %v", err)
			return
		}
	}
	if err := scanner.Err(); err != nil {
		log.Errorf("Error reading from pipe (%s): %v", level.String(), err)
	}
}

func pollForCommands(ctx context.Context, client pb.CommandServiceClient) {
	log.Info("Starting to poll for commands every 10 seconds...")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping Command polling.")
			return
		case <-ticker.C:
			ping := &pb.PingCommand{
				ClientId: *clientID,
				Time:     time.Now().Format(time.RFC3339Nano),
			}

			log.Debug("Pinging server for commands...")
			cmd, err := client.GetCommand(ctx, ping)
			if err != nil {
				log.Errorf("Failed to get Command from server: %v", err)
				continue
			}

			if cmd.NoCommandAvailable {
				log.Debug("No Command available from server.")
				continue
			}

			handleCommand(ctx, cmd)
		}
	}
}

func handleCommand(ctx context.Context, cmd *pb.Command) {
	log.Infof("Received Command: ID=%s, Type=%s, Target=%s, Payload=%s",
		cmd.CommandId, cmd.Type, cmd.TargetClient, cmd.Payload)

	switch cmd.Type {
	case pb.CommandType_PONG:
		log.Info("Received a PONG Command from the server.")

	case pb.CommandType_EXECUTE_SHELL:
		log.Info("Executing shell Command...")
		execCtx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()

		cmdParts := strings.Fields(cmd.Payload)
		shellCmd := exec.CommandContext(execCtx, cmdParts[0], cmdParts[1:]...)

		output, err := shellCmd.CombinedOutput()
		if err != nil {
			log.Errorf("Error executing shell Command '%s': %v. Output: %s", cmd.Payload, err, string(output))
		} else {
			log.Infof("Shell Command '%s' executed successfully. Output:\n%s", cmd.Payload, string(output))
		}

	case pb.CommandType_OPEN_SSH, pb.CommandType_OPEN_QEMU_MONITOR:
		log.Warnf("Received Command type %s, but interactive sessions are not yet implemented.", cmd.Type)
		// !TODO 在这里可以添加启动 TransportService 双向流的逻辑

	default:
		log.Warnf("Received unknown or unsupported Command type: %s", cmd.Type)
	}
}

//func main() {
//	flag.Parse()
//	log.SetLevel(log.InfoLevel)
//
//	// 1. 建立 gRPC 连接
//	log.Infof("Client '%s' starting, connecting to gRPC server at %s", *clientID, *ServerAddr)
//	conn, err := grpc.Dial(*ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
//	if err != nil {
//		log.Fatalf("Failed to connect: %v", err)
//	}
//	defer conn.Close()
//
//	// 2. 为所有服务创建客户端实例
//	logServiceClient := pb.NewLogStreamServiceClient(conn)
//	commandServiceClient := pb.NewCommandServiceClient(conn)
//	// transportServiceClient := pb.NewTransportServiceClient(conn) // 暂未使用
//
//	// 使用上下文来统一管理所有 goroutine 的生命周期
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	var wg sync.WaitGroup
//
//	// 3. 启动命令轮询 goroutine
//	wg.Add(1)
//	go func() {
//		defer wg.Done()
//		pollForCommands(ctx, commandServiceClient)
//	}()
//
//	// 4. 如果提供了初始命令，则启动日志上传 goroutine
//	if *Command != "" {
//		wg.Add(1)
//		go func() {
//			defer wg.Done()
//			executeAndStreamLogs(ctx, logServiceClient, *Command)
//		}()
//	}
//
//	log.Info("Client is running. Press Ctrl+C to exit.")
//	wg.Wait() // 等待所有核心任务结束
//	log.Info("Client shutting down.")
//}
