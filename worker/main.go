// Package worker is the muscle that does the work
// this file is just a testing placeholder
package worker

import (
	"fmt"
	"time"

	"github.com/ctfrancia/mongeta/task"
	"github.com/golang-collections/collections/queue"

	"github.com/google/uuid"
)

func main() {
	db := make(map[uuid.UUID]*task.Task)

	w := Worker{
		Queue: *queue.New(),
		DB:    db,
	}

	t := task.Task{
		ID:    uuid.New(),
		Name:  "test-container-1",
		State: task.Scheduled,
		Image: "strm/helloworld-http",
	}

	fmt.Println("starting task")
	w.AddTask(t)
	result := w.RunTask()
	if result.Error != nil {
		panic(result.Error)
	}

	t.ContainerID = result.ContainerID
	fmt.Printf("task %s is running in container %s\n", t.ID, t.ContainerID)
	fmt.Println("sleepy time")
	time.Sleep(time.Second * 30)

	fmt.Printf("stopping task %s\n", t.ID)
	t.State = task.Completed
	w.AddTask(t)
	result = w.RunTask()
	if result.Error != nil {
		panic(result.Error)
	}
}
