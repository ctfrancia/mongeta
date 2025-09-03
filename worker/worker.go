// Package worker is the muscle that does the work
// 1. Run tasks as a Docker container
// 2. Accept tasks to run from a manager
// 3. Procide relevant stats to the manager for the pupose of scheduling tasks
// 4. Keep track of its tasks and their state
package worker

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ctfrancia/mongeta/task"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	DB        map[uuid.UUID]*task.Task
	TaskCount int
}

func (w *Worker) CollectStats() {
	fmt.Println("I will Collect stats")
}

func (w *Worker) RunTask() task.DockerResult {
	t := w.Queue.Dequeue()
	if t == nil {
		log.Printf("No task in the queue")
		return task.DockerResult{Error: nil}
	}

	taskQueued, ok := t.(task.Task)
	if !ok {
		err := fmt.Errorf("invalid type %T", t)
		return task.DockerResult{Error: err}
	}

	taskPersisted := w.DB[taskQueued.ID]
	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.DB[taskQueued.ID] = &taskQueued

	}

	var result task.DockerResult
	if task.ValidStateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(taskQueued)
		case task.Completed:
			result = w.StopTask(taskQueued)
		default:
			result.Error = errors.New("we shouldn't get here")
		}
	} else {
		err := fmt.Errorf("invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		result.Error = err
	}

	return result
}

func (w *Worker) StartTask(t task.Task) task.DockerResult {
	t.StartTime = time.Now().UTC()

	config := task.NewConfig(&t)
	d := task.NewDocker(config)
	result := d.Run()
	if result.Error != nil {
		fmt.Printf("Error starting container %v: %v\n", t.ContainerID, result.Error)
		t.State = task.Failed
		w.DB[t.ID] = &t
		return result
	}

	t.ContainerID = result.ContainerID
	t.State = task.Running
	w.DB[t.ID] = &t

	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult {
	config := task.NewConfig(&t)

	d := task.NewDocker(config)

	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		fmt.Printf("Error stopping container %v: %v\n", t.ContainerID, result.Error)
		// return result
	}

	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.DB[t.ID] = &t
	log.Printf("Stopped and removed container %v for task %v\n", t.ContainerID, t.ID)

	return result
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}
