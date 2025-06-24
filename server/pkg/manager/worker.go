package manager

// memory worker manager; notice: sync with database

import (
	"Server/pkg/model"
	"gorm.io/gorm"
	"log/slog"
	"sync"
	"time"
)

type workerState struct {
	WorkerID string
	HostName string
	APIKey   string
	LastPing time.Time
}

type WorkerManager struct {
	db              *gorm.DB
	logger          *slog.Logger
	onlineWorkers   map[string]*workerState
	mu              sync.RWMutex
	timeout         time.Duration
	cleanupInterval time.Duration
}

func CreateWorkerManager(db *gorm.DB, logger *slog.Logger, timeout, cleanupInterval time.Duration) *WorkerManager {
	manager := &WorkerManager{
		db:              db,
		logger:          logger.With("component", "worker_manager"),
		onlineWorkers:   make(map[string]*workerState),
		timeout:         timeout,
		cleanupInterval: cleanupInterval,
	}

	go manager.cleanupLoop()

	return manager
}

func (m *WorkerManager) Register(workerID, hostname string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.onlineWorkers[workerID] = &workerState{
		WorkerID: workerID,
		HostName: hostname,
		LastPing: time.Now(),
	}
	m.logger.Info("worker registered in memory", "workerID", workerID)
}

func (m *WorkerManager) Ping(workerID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.onlineWorkers[workerID]
	if !exists {
		return false
	}
	state.LastPing = time.Now()
	return true
}

func (m *WorkerManager) IsOnline(workerID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.onlineWorkers[workerID]
	return exists
}

func (m *WorkerManager) cleanupLoop() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.cleanupTimedOutWorkers()
	}
}

func (m *WorkerManager) cleanupTimedOutWorkers() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for workerID, state := range m.onlineWorkers {
		if now.Sub(state.LastPing) > m.timeout {
			m.logger.Info("worker timed out, marking as offline", "workerID", workerID)

			delete(m.onlineWorkers, workerID)

			go func(id string) {
				err := m.db.Model(&model.Worker{}).Where("worker_id = ?", id).Update("status", "offline").Error
				if err != nil {
					m.logger.Error("Failed to update timed-out worker status in DB", "workerID", id, "error", err)
				}
			}(workerID)
		}
	}
}

func (m *WorkerManager) Unregister(workerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.onlineWorkers, workerID)
	m.logger.Info("worker unregistered in memory", "workerID", workerID)
}
