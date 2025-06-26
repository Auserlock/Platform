package model

import (
	"time"

	"github.com/google/uuid"
)

type TaskType string

const (
	TaskTypeKernelBuild TaskType = "kernel-build"
	TaskTypePatchApply  TaskType = "patch-apply"
)

type TaskStatus string

const (
	StatusPending TaskStatus = "pending"
	StatusRunning TaskStatus = "running"
	StatusSuccess TaskStatus = "success"
	StatusFailed  TaskStatus = "failed"
)

type Task struct {
	ID           uuid.UUID   `json:"id" gorm:"type:uuid;primary_key;"`
	Type         TaskType    `json:"type"`
	Status       TaskStatus  `json:"status"`
	Payload      CrashReport `json:"payload" gorm:"type:jsonb"`
	WorkerID     string      `json:"worker_id" gorm:"index"`
	Result       string      `json:"result"`
	ArtifactPath string      `json:"artifact_path"`
	ArtifactName string      `json:"artifact_name"`
	CreatedAt    time.Time   `json:"created_at"`
	StartedAt    *time.Time  `json:"started_at"`
	FinishedAt   *time.Time  `json:"finished_at"`
}

func CreateTask(taskType TaskType, payload CrashReport) *Task {
	return &Task{
		ID:        uuid.New(),
		Type:      taskType,
		Status:    StatusPending,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}
}
