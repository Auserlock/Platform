package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"worker/internal/config"
	queue "worker/internal/manage"
	"worker/internal/network"
	"worker/internal/parse"
	pb "worker/internal/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	log "github.com/sirupsen/logrus"
)

const (
	configFilePath       = "config/worker.json"
	queueName            = "task_queue"
	pingInterval         = time.Minute
	acceptTimeout        = 2 * time.Second
	reconnectDelay       = time.Minute
	maxReconnectAttempts = 5
)

var amqpURI string

// RegisterMsg 注册响应消息
type RegisterMsg struct {
	APIKey   string `json:"api_key"`
	Message  string `json:"message"`
	Status   string `json:"status"`
	WorkerID string `json:"worker_id"`
}

// Worker 工作节点配置
type Worker = config.Worker

// Message 任务消息
type Message struct {
	TaskID       string            `json:"id"`
	Type         string            `json:"type"`
	Status       string            `json:"status"`
	Payload      parse.CrashReport `json:"payload"`
	WorkerID     string            `json:"worker_id"`
	Result       string            `json:"result"`
	ArtifactPath string            `json:"artifact_path"`
	ArtifactName string            `json:"artifact_name"`
	CreatedAt    string            `json:"created_at"`
	StartedAt    string            `json:"started_at"`
	FinishedAt   string            `json:"finished_at"`
}

// WorkerService 工作节点服务
type WorkerService struct {
	worker      Worker
	client      *network.HttpClient
	rmqClient   *queue.RabbitMQClient // 修正：使用正确的类型名
	cancel      context.CancelFunc
	rmqMutex    sync.RWMutex
	isConnected bool
}

// NewWorkerService 创建工作节点服务
func NewWorkerService() (*WorkerService, error) {
	worker, err := config.LoadWorkerConfig(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load worker config: %w", err)
	}

	amqpURI = fmt.Sprintf("amqp://guest:123456@%s:5672/", config.GlobalWorker.IPAddress)

	config := network.Config{
		BaseURL: fmt.Sprintf("http://%s:8080", worker.IPAddress),
		Timeout: 30 * time.Second,
		Headers: map[string]string{},
	}
	client := network.NewClient(&config)

	return &WorkerService{
		worker: worker,
		client: client,
	}, nil
}

// saveWorkerConfig 保存工作节点配置
func (ws *WorkerService) saveWorkerConfig() error {
	data, err := json.MarshalIndent(ws.worker, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal worker config: %w", err)
	}

	if err := os.WriteFile(configFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// register 注册工作节点
func (ws *WorkerService) register(ctx context.Context) error {
	resp, err := ws.client.PostWithContext(ctx, "/api/v1/workers/register", ws.worker)
	if err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("register failed: %s", resp.String())
	}

	var registerMsg RegisterMsg
	if err := json.Unmarshal([]byte(resp.String()), &registerMsg); err != nil {
		return fmt.Errorf("failed to parse register response: %w", err)
	}

	log.Info(resp.String())
	ws.worker.APIKey = registerMsg.APIKey

	return ws.saveWorkerConfig()
}

// unregister 注销工作节点
func (ws *WorkerService) unregister(ctx context.Context) {
	if _, err := ws.client.PostWithContext(ctx, "/api/v1/workers/unregister", ws.worker); err != nil {
		log.Errorf("failed to unregister worker: %v", err)
	} else {
		log.Info("worker unregistered successfully")
	}
}

// setupRabbitMQ 初始化RabbitMQ连接
func (ws *WorkerService) setupRabbitMQ() error {
	rmqClient, err := queue.NewClient(amqpURI, queueName) // 修正：使用正确的函数名
	if err != nil {
		return fmt.Errorf("failed to create rabbitmq client: %w", err)
	}

	ws.rmqClient = rmqClient
	return nil
}

// handleMessage 处理任务消息
func (ws *WorkerService) handleMessage(ctx context.Context, messageBody string) error {
	var msg Message
	if err := json.Unmarshal([]byte(messageBody), &msg); err != nil {
		log.Errorf("failed to parse message JSON: %v", err)
		return fmt.Errorf("invalid message format: %w", err)
	}

	if !ws.checkFree() {
		log.Error("worker busy now, cannot accept task")
		return errors.New("worker busy")
	}

	if err := ws.acceptTask(ctx, msg.TaskID); err != nil {
		return fmt.Errorf("failed to accept task: %w", err)
	}

	time.Sleep(acceptTimeout)

	return ws.processTask(ctx, msg)
}

// acceptTask 接受任务
func (ws *WorkerService) acceptTask(ctx context.Context, taskID string) error {
	requestBody := map[string]string{
		"id":        taskID,
		"worker_id": ws.worker.WorkerID,
	}

	resp, err := ws.client.PostForm("/api/v1/tasks/accept", requestBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("accept task failed with status %d", resp.StatusCode)
	}

	return nil
}

// processTask 处理任务
func (ws *WorkerService) processTask(ctx context.Context, msg Message) error {
	log.Infof("client '%s' starting, connecting to gRPC server at %s", ws.worker.WorkerID, *network.ServerAddr)

	conn, err := grpc.NewClient(
		*network.ServerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                35 * time.Second,
			Timeout:             20 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("failed to close gRPC connection: %v", err)
		}
	}()

	tempFile, err := ws.createTempFile(msg.Payload)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer ws.cleanupTempFile(tempFile)

	report := parse.Parse(tempFile.Name())
	taskCtx := context.WithValue(ctx, "taskCommit", report.Crashes[0].KernelSourceCommit)
	taskCtx = context.WithValue(taskCtx, "taskID", msg.TaskID)
	taskCtx = context.WithValue(taskCtx, "workerID", ws.worker.WorkerID)

	logServiceClient := pb.NewLogStreamServiceClient(conn)
	command := fmt.Sprintf("../build-vmcore/kernel-builder -t %s -f %s -c -g -z", msg.Type, tempFile.Name())

	return network.ExecuteAndStreamLogs(taskCtx, logServiceClient, command, ws.client)
}

// createTempFile 创建临时文件
func (ws *WorkerService) createTempFile(payload parse.CrashReport) (*os.File, error) {
	payloadJSON, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload JSON: %w", err)
	}

	tempFile, err := os.CreateTemp("", "crash-report-*.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tempFile.Write(payloadJSON); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}

	return tempFile, nil
}

// cleanupTempFile 清理临时文件
func (ws *WorkerService) cleanupTempFile(tempFile *os.File) {
	if err := tempFile.Close(); err != nil {
		log.Errorf("failed to close temp file: %v", err)
	}

	if err := os.Remove(tempFile.Name()); err != nil {
		log.Errorf("failed to remove temp file: %v", err)
	}
}

// checkFree 检查工作节点是否空闲
func (ws *WorkerService) checkFree() bool {
	return true
}

// startMessageConsumer 启动消息消费者
func (ws *WorkerService) startMessageConsumer(ctx context.Context, wg *sync.WaitGroup) error {
	deliveries, err := ws.rmqClient.Consume()
	if err != nil {
		return fmt.Errorf("failed to consume messages: %w", err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer log.Info("message consumer stopped")

		for {
			select {
			case <-ctx.Done():
				log.Info("message consumer received cancel signal")
				return
			case d, ok := <-deliveries:
				if !ok {
					log.Info("message channel closed, exiting consumer")
					return
				}

				ws.processDelivery(ctx, d)
			}
		}
	}()

	return nil
}

// processDelivery 处理消息投递
func (ws *WorkerService) processDelivery(ctx context.Context, d queue.Delivery) { // 修正：使用正确的类型
	messageBody := string(d.Body)

	log.Infof("received new message: %s", messageBody)

	if err := ws.handleMessage(ctx, messageBody); err != nil {
		log.Warnf("task handling failed: %v, message: %s", err, messageBody)
		if err := d.Nack(true); err != nil {
			log.Errorf("failed to nack message: %v", err)
		}
	} else {
		log.Infof("task handled successfully, message: %s", messageBody)
		if err := d.Ack(); err != nil {
			log.Errorf("failed to ack message: %v", err)
		}
	}
}

// startHealthChecker 启动健康检查
func (ws *WorkerService) startHealthChecker(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer log.Info("health checker stopped")

		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("health checker received cancel signal")
				return
			case <-ticker.C:
				ws.ping(ctx)
			}
		}
	}()
}

// ping 发送心跳
func (ws *WorkerService) ping(ctx context.Context) {
	log.Debug("sending ping request")

	resp, err := ws.client.PostWithContext(ctx, "/api/v1/workers/ping", nil)
	if err != nil {
		if !errors.Is(ctx.Err(), context.Canceled) {
			log.Errorf("ping failed: %v", err)
		}
		return
	}

	if !resp.IsSuccess() {
		log.Errorf("ping failed with status %d: %s", resp.StatusCode, resp.String())
	}
}

// Run 运行工作节点服务
func (ws *WorkerService) Run() error {
	initLogger()

	ctx, cancel := context.WithCancel(context.Background())
	ws.cancel = cancel
	defer cancel()

	// 注册工作节点
	if err := ws.register(ctx); err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}

	// 设置认证头
	authHeader := fmt.Sprintf("Bearer %s:%s", ws.worker.WorkerID, ws.worker.APIKey)
	ws.client.SetHeader("Authorization", authHeader)

	// 初始化RabbitMQ
	if err := ws.setupRabbitMQ(); err != nil {
		return fmt.Errorf("rabbitmq setup failed: %w", err)
	}
	defer func() {
		if err := ws.rmqClient.Close(); err != nil {
			log.Errorf("failed to close rabbitmq client: %v", err)
		}
	}()

	var wg sync.WaitGroup

	// 启动消息消费者
	if err := ws.startMessageConsumer(ctx, &wg); err != nil {
		return fmt.Errorf("failed to start message consumer: %w", err)
	}

	// 启动健康检查
	ws.startHealthChecker(ctx, &wg)

	// 等待退出信号
	ws.waitForShutdown()

	log.Info("shutting down worker service...")
	ws.unregister(ctx)
	cancel()
	wg.Wait()

	log.Info("worker service stopped gracefully")
	return nil
}

// waitForShutdown 等待关闭信号
func (ws *WorkerService) waitForShutdown() {
	log.Info("worker service started. Press CTRL-C to exit")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("received shutdown signal")
}

// initLogger 初始化日志配置
func initLogger() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       false,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: true,
		QuoteEmptyFields:       false,
		DisableQuote:           true,
		ForceColors:            true,
	})
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
}

func main() {
	workerService, err := NewWorkerService()
	if err != nil {
		log.Fatalf("failed to create worker service: %v", err)
	}

	if err := workerService.Run(); err != nil {
		log.Fatalf("worker service failed: %v", err)
	}
}
