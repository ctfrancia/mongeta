package manager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ctfrancia/mongeta/logger"
	"github.com/ctfrancia/mongeta/task"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (a *API) StartTaskHandler(w http.ResponseWriter, r *http.Request) {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()

	te := task.TaskEvent{}
	err := d.Decode(&te)
	if err != nil {
		msg := fmt.Sprintf("Error unmarshalling task event: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		e := ErrResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        msg,
		}
		json.NewEncoder(w).Encode(e)
		return
	}

	a.Manager.AddTask(te)
	logger.Info("added task to manager", "task_id", te.Task.ID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(te.Task)
}

func (m *Manager) GetTasks() []*task.Task {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tasks := make([]*task.Task, 0, len(m.TaskDB))
	for _, t := range m.TaskDB {
		tasks = append(tasks, t)
	}
	return tasks
}

func (a *API) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Manager.GetTasks())
}

func (a *API) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		logger.Warn("no taskID in request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tID, err := uuid.Parse(taskID)
	if err != nil {
		logger.Warn("invalid taskID", "task_id", taskID, "err", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	a.Manager.mu.RLock()
	taskToStop, ok := a.Manager.TaskDB[tID]
	a.Manager.mu.RUnlock()
	if !ok {
		logger.Warn("task not found", "task_id", tID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Completed,
		TimeStamp: time.Now(),
	}

	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	te.Task = taskCopy
	a.Manager.AddTask(te)

	logger.Info("stopping task", "task_id", taskToStop.ID)
	w.WriteHeader(http.StatusNoContent)
}
