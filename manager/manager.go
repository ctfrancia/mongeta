// Package manager handles the schedule/ api/stask storage/ workers
// 1. Accept requests from users to start and stop tasks
// 2. Schedule tasks onto worker machines
// 3. Keep track of tasks, their states, and the machine on which they run
package manager

import (
	"fmt"

	"github.com/ctfrancia/mongeta/task"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Manager struct {
	Pending       queue.Queue
	TaskDB        map[string][]*task.Task
	EventDB       map[string][]*task.TaskEvent
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
}

func (m *Manager) SelectWorker() {
	fmt.Println("I will select an appropriate worker")
}

func (m *Manager) UpdateTasks() {
	fmt.Println("I will update the tasks")
}

func (m *Manager) SendWork() {
	fmt.Println("I will send work to workers")
}
