package rpc

import (
	pb "Server/pkg/proto"
	"Server/pkg/websocket"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type LogMessageForJSON struct {
	Time    string `json:"time"`
	Message string `json:"message"`
	TaskID  string `json:"taskId"`
}

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
	log.Infoln("start upload logs")
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
			logMsg.TaskId,
			logMsg.Message)

		if s.Hub != nil {
			logForWs := LogMessageForJSON{
				TaskID:  logMsg.TaskId,
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
