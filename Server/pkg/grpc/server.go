package rpc

import (
	pb "Server/pkg/proto"
	"Server/pkg/websocket" // 1. 【已添加】导入我们创建的 websocket 包
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"sync"
	"time"
)

// LogMessageForJSON 定义了要发送给 WebSocket 客户端的JSON结构
// 我们需要这个结构来确保字段名是小驼峰 (camelCase)，以匹配前端的期望
type LogMessageForJSON struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
	TaskID  string `json:"taskId"` // 注意这里的字段名是 "taskId"
}

// LogStreamServer 结构体
type LogStreamServer struct {
	pb.UnimplementedLogStreamServiceServer
	mu   sync.RWMutex
	logs []*pb.LogMessage

	Hub *websocket.Hub
}

func NewLogStreamServer(hub *websocket.Hub) *LogStreamServer {
	return &LogStreamServer{
		logs: make([]*pb.LogMessage, 0),
		Hub:  hub,
	}
}

func (s *LogStreamServer) UploadLogs(stream pb.LogStreamService_UploadLogsServer) error {
	log.Infoln("Start upload logs")
	var count int64
	var client string

	for {
		logMsg, err := stream.Recv()
		if err == io.EOF {
			log.Infof("client %s finished uploading %d logs", client, count)
			return stream.SendAndClose(&pb.UploadLogsResponse{
				Success: true,
				Message: fmt.Sprintf("successfully received %d logs from client %s", count, client),
			})
		}
		if err != nil {
			log.Infof("Error receiving log: %v", err)
			return err
		}

		s.mu.Lock()
		s.logs = append(s.logs, logMsg)
		s.mu.Unlock()

		client = logMsg.ClientId
		count++

		log.Infof("Received log from %s [%s]: %s",
			logMsg.ClientId,
			logMsg.Level.String(),
			logMsg.Message)

		if s.Hub != nil {
			logForWs := LogMessageForJSON{
				TaskID:  logMsg.ClientId,
				Time:    time.Now().Format(time.RFC3339),
				Message: logMsg.Message,
			}

			msgBytes, jsonErr := json.Marshal(logForWs)
			if jsonErr != nil {
				log.Errorf("Failed to marshal log message for WebSocket: %v", jsonErr)
				continue
			}

			select {
			case s.Hub.Broadcast <- msgBytes:
			default:
				log.Warnf("WebSocket broadcast channel is full. Log message from %s dropped.", logMsg.ClientId)
			}
		}
	}
}

func (s *LogStreamServer) GetStoredLogs() []*pb.LogMessage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logs := make([]*pb.LogMessage, len(s.logs))
	copy(logs, s.logs)
	return logs
}
