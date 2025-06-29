package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Worker struct {
	WorkerID  string `json:"worker_id"`
	Hostname  string `json:"hostname"`
	APIKey    string `json:"api_key"`
	IPAddress string `json:"ip_address"`
}

var GlobalWorker Worker

// LoadWorkerConfig 加载工作节点配置
func LoadWorkerConfig(filePath string) (Worker, error) {
	jsonData, err := os.ReadFile(filePath)
	if err != nil {
		return GlobalWorker, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(jsonData, &GlobalWorker); err != nil {
		return GlobalWorker, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return GlobalWorker, nil
}
