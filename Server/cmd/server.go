package main

import (
	"Server/pkg/grpc"
	"Server/pkg/manager"
	pb "Server/pkg/proto"
	"Server/pkg/router"
	"Server/pkg/websocket"
	"github.com/lmittmann/tint"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"log/slog"
	"net"
	"os"
	"time"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       false,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: true,
		QuoteEmptyFields:       false,
		DisableQuote:           true,
		ForceColors:            true,
	})

	log.SetLevel(log.DebugLevel)

	tintOptions := &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: "2006-01-02 15:04:05",
		AddSource:  true,
	}

	handler := tint.NewHandler(os.Stdout, tintOptions)
	logger := slog.New(handler)

	slog.SetDefault(logger)
}

func main() {
	wsHub := websocket.NewHub()
	go wsHub.Run()

	go func() {
		port := ":50051"
		lis, err := net.Listen("tcp", port)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		gRPCServer := grpc.NewServer()
		logServer := rpc.NewLogStreamServer(wsHub)
		pb.RegisterLogStreamServiceServer(gRPCServer, logServer)
		if err := gRPCServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	manager.Init()

	workerTimeout := 2 * time.Minute
	cleanupInterval := 30 * time.Second

	amqpURI := "amqp://guest:123456@localhost:5672/"
	queueName := "task_queue"
	rmqClient, err := manager.CreateRabbitMQClient(amqpURI, queueName, slog.Default())
	if err != nil {
		return
	}
	defer func() {
		if err := rmqClient.Close(); err != nil {
			return
		}
	}()

	workerMgr := manager.CreateWorkerManager(manager.DB, slog.Default(), workerTimeout, cleanupInterval)

	r := router.SetupRouter(rmqClient, manager.DB, workerMgr, wsHub)
	err = r.Run(":8080")
	if err != nil {
		slog.Error(err.Error())
		panic(err)
		return
	}
}
