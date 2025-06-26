package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Port string   `json:"port"` // proxy port
	VM   VMConfig `json:"vm"`
}

type VMConfig struct {
	Memory string `json:"memory"` // memory size in MB
}

var GlobalConfig Config

func Load(file string) error {
	data, err := os.ReadFile(file)

	if err != nil {
		return fmt.Errorf("failed to read config file %s: %v", file, err)
	}

	err = json.Unmarshal(data, &GlobalConfig)

	if err != nil {
		return fmt.Errorf("failed to parse config file %s: %v", file, err)
	}

	return nil
}
