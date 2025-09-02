// Package worker is the muscle that does the work
// 1. Run tasks as a Docker container
// 2. Accept tasks to run from a manager
// 3. Procide relevant stats to the manager for the pupose of scheduling tasks
// 4. Keep track of its tasks and their state
package worker

import (
	"fmt"

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

func (w *Worker) RunTask() {
	fmt.Println("I will start or stop a task")
}

func (w *Worker) StartTask() {
	fmt.Println("I will start a task")
}

func (w *Worker) StopTask() {
	fmt.Println("I will stop a task")
}
