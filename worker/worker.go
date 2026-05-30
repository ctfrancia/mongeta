// Package worker is the muscle that does the work
// 1. Run tasks as a Docker container
// 2. Accept tasks to run from a manager
// 3. Provide relevant stats to the manager for the purpose of scheduling tasks
// 4. Keep track of its tasks and their state
package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ctfrancia/mongeta/logger"
	"github.com/ctfrancia/mongeta/task"
	"github.com/google/uuid"
)

type Worker struct {
	Name      string
	Queue     chan task.Task
	DB        map[uuid.UUID]*task.Task
	mu        sync.RWMutex
	Stats     *Stats
	TaskCount int
}

func NewWorker(queueSize int) *Worker {
	return &Worker{
		Queue: make(chan task.Task, queueSize),
		DB:    make(map[uuid.UUID]*task.Task),
	}
}

func (w *Worker) GetTasks() []*task.Task {
	w.mu.RLock()
	defer w.mu.RUnlock()
	tasks := make([]*task.Task, 0, len(w.DB))
	for _, t := range w.DB {
		tasks = append(tasks, t)
	}
	return tasks
}

// GetTask returns the task with the given ID and whether it was found,
// reading w.DB under a read lock.
func (w *Worker) GetTask(id uuid.UUID) (*task.Task, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	t, ok := w.DB[id]
	return t, ok
}

func (w *Worker) CollectStats(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Info("collecting stats")
			w.Stats = GetStats()
			w.Stats.TaskCount = w.TaskCount
			logger.Info("stats collected")
		}
	}
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	t.StartTime = time.Now().UTC()

	config := task.NewConfig(&t)
	d, err := task.NewDocker(config)
	if err != nil {
		logger.Error("error creating docker client", "err", err)
		return task.DockerResult{Error: err}
	}

	result := d.Run()
	if result.Error != nil {
		logger.Error("error starting container", "container_id", t.ContainerID, "err", result.Error)
		t.State = task.Failed
		w.mu.Lock()
		w.DB[t.ID] = &t
		w.mu.Unlock()
		return result
	}

	t.ContainerID = result.ContainerID
	t.State = task.Running
	w.mu.Lock()
	w.DB[t.ID] = &t
	w.mu.Unlock()

	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	config := task.NewConfig(&t)

	d, err := task.NewDocker(config)
	if err != nil {
		logger.Error("error creating docker client", "err", err)
		return task.DockerResult{Error: err}
	}

	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		logger.Error("error stopping container", "container_id", t.ContainerID, "err", result.Error)
		return result
	}

	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.mu.Lock()
	w.DB[t.ID] = &t
	w.mu.Unlock()
	logger.Info("stopped container", "container_id", t.ContainerID, "task_id", t.ID)

	return result
}

func (w *Worker) AddTask(t task.Task) {
	select {
	case w.Queue <- t:
	default:
		logger.Warn("worker queue full, dropping task", "task_id", t.ID)
	}
}

func (w *Worker) runTask() task.DockerResult {
	select {
	case taskQueued := <-w.Queue:
		w.mu.RLock()
		taskPersisted := w.DB[taskQueued.ID]
		w.mu.RUnlock()

		if taskPersisted == nil {
			taskPersisted = &taskQueued
			w.mu.Lock()
			w.DB[taskQueued.ID] = &taskQueued
			w.mu.Unlock()
		}

		var result task.DockerResult
		if task.ValidStateTransition(taskPersisted.State, taskQueued.State) {
			switch taskQueued.State {
			case task.Scheduled:
				result = w.StartTask(taskQueued)
			case task.Completed:
				result = w.StopTask(taskQueued)
			default:
				result.Error = errors.New("we should not get here")
			}
		} else {
			result.Error = fmt.Errorf("invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		}
		return result
	default:
		logger.Debug("no tasks in queue")
		return task.DockerResult{Error: nil}
	}
}

func (w *Worker) InspectTask(t task.Task) task.DockerInspectResponse {
	config := task.NewConfig(&t)
	d, err := task.NewDocker(config)
	if err != nil {
		logger.Error("error creating docker client", "err", err)
		return task.DockerInspectResponse{Error: err}
	}

	return d.Inspect(context.Background(), t.ContainerID)
}

func (w *Worker) UpdateTasks(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger.Info("checking task status")
			w.updateTasks()
			logger.Info("task updates completed")
		}
	}
}

func (w *Worker) updateTasks() {
	w.mu.RLock()
	ids := make([]uuid.UUID, 0, len(w.DB))
	for id, t := range w.DB {
		if t.State == task.Running {
			ids = append(ids, id)
		}
	}
	w.mu.RUnlock()

	for _, id := range ids {
		w.mu.RLock()
		t := w.DB[id]
		w.mu.RUnlock()

		resp := w.InspectTask(*t)
		if resp.Error != nil {
			logger.Error("error updating task", "task_id", id, "err", resp.Error)
		}

		w.mu.Lock()
		if resp.Container == nil {
			logger.Error("no container for running task", "task_id", id)
			w.DB[id].State = task.Failed
			w.mu.Unlock()
			continue
		}

		if resp.Container.State.Status == "exited" {
			logger.Warn("container in non-running state", "task_id", id, "status", resp.Container.State.Status)
			w.DB[id].State = task.Failed
		}

		w.DB[id].HostPorts = resp.Container.NetworkSettings.NetworkSettingsBase.Ports
		w.mu.Unlock()
	}
}

func (w *Worker) RunTasks(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result := w.runTask()
			if result.Error != nil {
				logger.Error("error running task", "err", result.Error)
			}
		}
	}
}
