package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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
		e := ErrorResponse{
			HTTPStatusCode: http.StatusBadRequest,
			Message:        msg,
		}
		json.NewEncoder(w).Encode(e)
		return
	}

	a.Worker.AddTask(te.Task)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(te.Task)
}

func (a *API) GetTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Worker.GetTasks())
}

func (a *API) StopTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		log.Printf("No taskID passed in request.\n")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tID, err := uuid.Parse(taskID)
	if err != nil {
		log.Printf("Invalid taskID %q: %v\n", taskID, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	taskToStop, ok := a.Worker.GetTask(tID)
	if !ok {
		log.Printf("No task with ID %v found", tID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	taskCopy := *taskToStop
	taskCopy.State = task.Completed
	a.Worker.AddTask(taskCopy)

	log.Printf("Added task %v to stop container %v\n", taskToStop.ID,
		taskToStop.ContainerID)

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) GetStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(a.Worker.Stats)
}
