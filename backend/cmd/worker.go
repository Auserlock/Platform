package main

import (
	"backend/internal/manage"
	"backend/internal/network"
	pb "backend/internal/proto"
	"backend/pkg/parse"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

type RegisterMsg struct {
	APIKey   string `json:"api_key"`
	Message  string `json:"message"`
	Status   string `json:"status"`
	WorkerID string `json:"worker_id"`
}

type Worker struct {
	WorkerID string `json:"worker_id"`
	Hostname string `json:"hostname"`
	APIKey   string `json:"api_key"`
}

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

var worker Worker

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

	configFilePath := "config/worker.json"
	jsonData, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Infof("failed to read config file: %v\n", err)
		return
	}

	err = json.Unmarshal(jsonData, &worker)
	if err != nil {
		log.Infof("failed to parse json: %v\n", err)
		return
	}
}

func handleMsg(message string, client *network.HttpClient) error {
	var msg Message
	err := json.Unmarshal([]byte(message), &msg)
	if err != nil {
		log.Errorf("failed to parse json: %v\n", err)
		os.Exit(1)
	}

	if !checkFree() {
		log.Errorf("worker busy now. cannot accept task, sending nack()")
		return errors.New("worker busy now")
	}

	requestBody := map[string]string{
		"id":        msg.TaskID,
		"worker_id": worker.WorkerID,
	}
	resp, err := client.PostForm("/api/v1/tasks/accept", requestBody)
	if err != nil {
		log.Errorf("failed to accept task: %v\n", err)
		return err
	}
	if resp.StatusCode != 200 {
		log.Errorf("failed to accept task: %v\n", resp)
		return errors.New("failed to accept task")
	}
	//log.Infoln(resp.String())

	time.Sleep(time.Second * 2)

	log.Infof("client '%s' starting, connecting to gRPC server at %s", worker.WorkerID, *network.ServerAddr)
	conn, err := grpc.NewClient(*network.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Errorf("failed to close connection: %v", err)
		}
	}()

	logServiceClient := pb.NewLogStreamServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	payloadJSON, err := json.MarshalIndent(msg.Payload, "", "  ")
	if err != nil {
		log.Errorf("failed marshal JSON: %v", err)
		return err
	}

	tempFile, err := os.CreateTemp("", "crash-report-*.json")
	if err != nil {
		log.Errorf("failed create temp file: %v", err)
		return err
	}

	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil {
			log.Errorf("failed remove temp file: %v", err)
		}
	}()
	defer func() {
		if err := tempFile.Close(); err != nil {
			log.Errorf("failed close temp file: %v", err)
		}
	}()

	_, err = tempFile.Write(payloadJSON)
	if err != nil {
		log.Errorf("failed writting data to temp file: %v", err)
		return err
	}

	report := parse.Parse(tempFile.Name())
	ctx = context.WithValue(ctx, "taskCommit", report.Crashes[0].KernelSourceCommit)

	command := fmt.Sprintf("./kernel-builder %s %s", msg.Type, tempFile.Name())
	ctx = context.WithValue(ctx, "taskID", msg.TaskID)
	err = network.ExecuteAndStreamLogs(ctx, logServiceClient, command, client)
	if err != nil {
		return err
	}

	return nil
}

func checkFree() bool {
	return true
}

func main() {
	config := network.Config{
		BaseURL: "http://localhost:8080",
		Timeout: time.Second * 10,
		Headers: map[string]string{},
	}
	client := network.NewClient(&config)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	resp, err := client.PostWithContext(ctx, "/api/v1/workers/register", worker)
	if err != nil {
		log.Errorf("failed to register worker: %v\n", err)
		os.Exit(1)
	}
	if !resp.IsSuccess() {
		log.Errorf("register failed: %s\n", resp.String())
		os.Exit(1)
	}
	fmt.Println(resp.String())
	var registerMsg RegisterMsg
	err = json.Unmarshal([]byte(resp.String()), &registerMsg)
	if err != nil {
		os.Exit(1)
	}
	defer func() {
		_, err := client.PostWithContext(ctx, "/api/v1/workers/unregister", worker)
		if err != nil {
			return
		}
	}()

	configFilePath := "config/worker.json"
	worker.APIKey = registerMsg.APIKey
	newWorker, _ := json.Marshal(worker)
	err = os.WriteFile(configFilePath, newWorker, 0644)
	if err != nil {
		return
	}

	amqpURI := "amqp://guest:123456@localhost:5672/"
	queueName := "task_queue"

	rmqClient, err := queue.NewClient(amqpURI, queueName)
	if err != nil {
		log.Errorf("failed to create rabbitmq client, error: %v", err)
		os.Exit(1)
	}

	authHeader := fmt.Sprintf("Bearer %s:%s", worker.WorkerID, worker.APIKey)
	client.SetHeader("Authorization", authHeader)

	deliveries, err := rmqClient.Consume()
	if err != nil {
		log.Errorf("failed to consume, error: %v", err)
		os.Exit(1)
	}

	wg.Add(1)

	go func() {
		defer wg.Done()
		for d := range deliveries {
			log.Infof("get a new message: body: %s", string(d.Body))

			err := handleMsg(string(d.Body), client)

			if err != nil {
				log.Warnf("task handling failed..., body: %s", string(d.Body))
				if err := d.Nack(true); err != nil {
					log.Errorf("Nack message failed, error: %v", err)
				}
			} else {
				log.Infof("task handle successfully, sending ack manully, body: %s", string(d.Body))
				if err := d.Ack(); err != nil {
					log.Errorf("Ack message failed, error: %v", err)
				}
			}
		}

		log.Infoln("message channel closed, exit normally")
	}()

	wg.Add(1)

	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Infoln("received cancel signal, quitting")
				return
			case <-ticker.C:
				log.Infoln("sending ping request")
				pingResp, err := client.PostWithContext(ctx, "/api/v1/workers/ping", nil)
				if err != nil {
					if !errors.Is(ctx.Err(), context.Canceled) {
						log.Errorf("ping failed: %v", err)
					}
					continue
				}
				if !pingResp.IsSuccess() {
					log.Errorf("failed to ping: %d, response: %s", pingResp.StatusCode, pingResp.String())
					continue
				}
			}
		}
	}()

	log.Infoln("program running. press CTRL-C to exit")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Infoln("shutting down rmq client...")
	if err := rmqClient.Close(); err != nil {
		log.Errorf("failed to close rabbitmq client, error: %v", err)
		os.Exit(1)
	}

	log.Infoln("received quit signal, shutting down")
	cancel()

	wg.Wait()
	log.Infoln("all tasks done. quitting down")
}
