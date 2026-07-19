// Package manager handles the schedule/ api/task storage/ workers
// 1. Accept requests from users to start and stop tasks
// 2. Schedule tasks onto worker machines
// 3. Keep track of tasks, their states, and the machine on which they run
package manager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ctfrancia/mongeta/logger"
	"github.com/ctfrancia/mongeta/task"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type Manager struct {
	Pending       chan task.TaskEvent
	TaskDB        map[uuid.UUID]*task.Task
	EventDB       map[uuid.UUID]*task.TaskEvent
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	LastWorker    int
	MaxRestarts   int
	mu            sync.RWMutex
}

func New(workers []string, queueSize int, maxRestarts int) *Manager {
	taskDB := make(map[uuid.UUID]*task.Task)
	eventDB := make(map[uuid.UUID]*task.TaskEvent)
	workerTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)
	for worker := range workers {
		workerTaskMap[workers[worker]] = []uuid.UUID{}
	}

	return &Manager{
		Pending:       make(chan task.TaskEvent, queueSize),
		TaskDB:        taskDB,
		EventDB:       eventDB,
		Workers:       workers,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
		MaxRestarts:   maxRestarts,
	}
}

func (m *Manager) SelectWorker() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var newWorker int
	if m.LastWorker+1 < len(m.Workers) {
		newWorker = m.LastWorker + 1
		m.LastWorker++
	} else {
		newWorker = 0
		m.LastWorker = 0
	}

	return m.Workers[newWorker]
}

func (m *Manager) SendWork() {
	select {
	case te := <-m.Pending:
		w := m.SelectWorker()
		t := te.Task

		logger.Info("sending task to worker", "task_id", t.ID, "worker", w)

		m.mu.Lock()
		m.EventDB[te.ID] = &te
		m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], te.Task.ID)
		m.TaskWorkerMap[t.ID] = w
		t.State = task.Scheduled
		m.TaskDB[t.ID] = &t
		m.mu.Unlock()

		data, err := json.Marshal(te)
		if err != nil {
			logger.Error("unable to marshal task", "task_id", t.ID, "err", err)
		}

		url := fmt.Sprintf("http://%s/tasks", w)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			logger.Error("error sending task to worker", "worker", w, "err", err)
			return
		}
		d := json.NewDecoder(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			e := ErrResponse{}
			err := d.Decode(&e)
			if err != nil {
				logger.Error("error decoding error response", "err", err)
				return
			}
			logger.Error("worker rejected task", "status", e.HTTPStatusCode, "message", e.Message)
			return
		}

		t = task.Task{}
		err = d.Decode(&t)
		if err != nil {
			logger.Error("error decoding task response", "err", err)
			return
		}
		logger.Debug("task confirmed by worker", "task_id", t.ID)
	default:
		logger.Debug("no work in queue")
	}
}

func (m *Manager) AddTask(te task.TaskEvent) {
	select {
	case m.Pending <- te:
	default:
		logger.Warn("manager queue full, dropping task", "task_id", te.Task.ID)
	}
}

func (m *Manager) UpdateTasks(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Info("checking task updates from workers")
			m.updateTasks()
			logger.Info("task updates completed")
		}
	}
}

func (m *Manager) updateTasks() {
	for _, worker := range m.Workers {
		logger.Info("checking worker for task updates", "worker", worker)
		url := fmt.Sprintf("http://%s/tasks", worker)
		resp, err := http.Get(url)
		if err != nil {
			logger.Error("error connecting to worker", "worker", worker, "err", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			logger.Error("unexpected status from worker", "worker", worker, "status", resp.StatusCode)
			continue
		}

		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task
		err = d.Decode(&tasks)
		if err != nil {
			logger.Error("error decoding tasks from worker", "worker", worker, "err", err)
		}

		for _, t := range tasks {
			logger.Debug("updating task", "task_id", t.ID)

			m.mu.Lock()
			_, ok := m.TaskDB[t.ID]
			if !ok {
				logger.Warn("task not found in local db", "task_id", t.ID)
				m.mu.Unlock()
				continue
			}
			if m.TaskDB[t.ID].State != t.State {
				m.TaskDB[t.ID].State = t.State
			}
			m.TaskDB[t.ID].StartTime = t.StartTime
			m.TaskDB[t.ID].FinishTime = t.FinishTime
			m.TaskDB[t.ID].ContainerID = t.ContainerID
			m.mu.Unlock()
		}
	}
}

// GetTaskWorker returns the worker address assigned to the given task ID and
// whether an assignment exists, reading m.TaskWorkerMap under a read lock.
func (m *Manager) GetTaskWorker(id uuid.UUID) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	w, ok := m.TaskWorkerMap[id]
	return w, ok
}

func (m *Manager) checkTaskHealth(t task.Task) error {
	logger.Info("calling health check", "task_id", t.ID, "endpoint", t.HealthCheck)
	w, ok := m.GetTaskWorker(t.ID)
	if !ok {
		logger.Warn("no worker assigned to task", "task_id", t.ID)
		return fmt.Errorf("no worker assigned to task %s", t.ID)
	}
	hostPort := getHostPort(t.HostPorts)
	worker := strings.Split(w, ":")
	url := fmt.Sprintf("http://%s:%s/%s", worker[0], *hostPort, t.HealthCheck)

	logger.Debug("health check url", "task_id", t.ID, "url", url)
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("health check connection failed", "url", url, "err", err)
		return fmt.Errorf("error connecting to health check %s", url)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Warn("health check returned non-200", "task_id", t.ID, "status", resp.StatusCode)
		return fmt.Errorf("health check for task %s did not return 200", t.ID)
	}

	logger.Info("health check passed", "task_id", t.ID, "status", resp.StatusCode)
	return nil
}

func (m *Manager) DoHealthChecks(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Info("performing health checks")
			m.doHealthChecks()
			logger.Info("health checks completed")
		}
	}
}

func (m *Manager) doHealthChecks() {
	for _, t := range m.GetTasks() {
		if t.State == task.Running && t.RestartCount < m.MaxRestarts {
			if err := m.checkTaskHealth(*t); err != nil {
				m.restartTask(t)
			}
		} else if t.State == task.Failed && t.RestartCount < m.MaxRestarts {
			m.restartTask(t)
		}
	}
}

func (m *Manager) restartTask(t *task.Task) {
	m.mu.Lock()
	t.RestartCount++
	t.State = task.Scheduled
	m.TaskDB[t.ID] = t
	m.mu.Unlock()

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Scheduled,
		TimeStamp: time.Now(),
		Task:      *t,
	}
	m.AddTask(te)
	logger.Info("restarting task", "task_id", t.ID, "attempt", t.RestartCount)
}

func getHostPort(ports nat.PortMap) *string {
	for k := range ports {
		return &ports[k][0].HostPort
	}
	return nil
}

func (m *Manager) ProcessTasks(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Info("processing tasks")
			m.SendWork()
		}
	}
}
