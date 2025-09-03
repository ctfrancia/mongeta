package main

import (
	"fmt"
	//"os"
	"time"

	//"github.com/ctfrancia/mongeta/manager"
	//"github.com/ctfrancia/mongeta/node"
	"github.com/ctfrancia/mongeta/task"
	"github.com/ctfrancia/mongeta/worker"

	"github.com/golang-collections/collections/queue"

	"github.com/google/uuid"
	"github.com/moby/moby/client"
)

func main() {
	db := make(map[uuid.UUID]*task.Task)

	w := worker.Worker{
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

	/*j
	t := task.Task{
		ID:     uuid.New(),
		Name:   "task-1",
		State:  task.Pending,
		Image:  "image-1",
		Memory: 1024,
		Disk:   1,
	}

	te := task.TaskEvent{
		ID:        uuid.New(),
		State:     task.Pending,
		TimeStamp: time.Now(),
		Task:      t,
	}

	fmt.Printf("task: %v\n", t)
	fmt.Printf("task event: %v\n", te)

	w := worker.Worker{
		Name:  "worker-1",
		Queue: *queue.New(),
		DB:    make(map[uuid.UUID]*task.Task),
	}

	fmt.Printf("worker: %v\n", w)
	w.CollectStats()
	w.RunTask()
	w.StartTask()
	w.StopTask()

	m := manager.Manager{
		Pending: *queue.New(),
		TaskDB:  make(map[string][]*task.Task),
		EventDB: make(map[string][]*task.TaskEvent),
		Workers: []string{w.Name},
	}

	fmt.Printf("manager: %v\n", m)
	m.SelectWorker()
	m.UpdateTasks()
	m.SendWork()

	n := node.Node{
		Name:   "node-1",
		IP:     "192.168.1.1",
		Cores:  4,
		Memory: 1024,
		Disk:   25,
		Role:   "worker",
	}

	fmt.Printf("node: %v\n", n)

	fmt.Printf("Create a test container\n")
	dockerTest, createResult := createContainer()
	if createResult.Error != nil {
		fmt.Printf("Error: %v\n", createResult.Error)
		os.Exit(1)
	}
	time.Sleep(time.Second * 5)
	fmt.Printf("stopping container %s\n", createResult.ContainerID)
	_ = stopContainer(dockerTest, createResult.ContainerID)
	*/
}

func createContainer() (*task.Docker, *task.DockerResult) {
	c := task.Config{
		Name:  "test-container-1",
		Image: "postgres:13",
		Env: []string{
			"POSTGRES_USER=mongeta",
			"POSTGRES_PASSWORD=secret",
		},
	}

	dc, _ := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	d := task.Docker{
		Client: dc,
		Config: c,
	}
	result := d.Run()
	if result.Error != nil {
		fmt.Printf("Error: %v\n", result.Error)
		return nil, nil
	}

	fmt.Printf("Container %s is running with config %v\n", result.ContainerID, c)
	return &d, &result
}

func stopContainer(d *task.Docker, ID string) *task.DockerResult {
	result := d.Stop(ID)
	if result.Error != nil {
		fmt.Printf("Error: %v\n", result.Error)
		return nil
	}
	fmt.Printf("Container %s has been stopped and removed\n", result.ContainerID)
	return &result
}
