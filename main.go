package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ctfrancia/mongeta/manager"
	"github.com/ctfrancia/mongeta/task"
	"github.com/ctfrancia/mongeta/worker"

	"github.com/golang-collections/collections/queue"

	"github.com/google/uuid"
	"github.com/moby/moby/client"
)

func main() {
	whost := os.Getenv("MONGETA_WORKER_HOST")
	wport, _ := strconv.Atoi(os.Getenv("MONGETA_WORKER_PORT"))

	mhost := os.Getenv("MONGETA_HOST")
	mport, _ := strconv.Atoi(os.Getenv("MONGETA_PORT"))

	fmt.Println("Starting Mongeta worker")

	w := worker.Worker{
		Queue: *queue.New(),
		DB:    make(map[uuid.UUID]*task.Task),
	}
	wapi := worker.API{Address: whost, Port: wport, Worker: &w}

	go w.RunTasks()
	go w.CollectStats()
	go wapi.Start()

	workers := []string{fmt.Sprintf("%s:%d", whost, wport)}
	m := manager.New(workers)
	mapi := manager.API{Address: mhost, Port: mport, Manager: m}
	go m.ProcessTasks()
	go m.UpdateTasks()

	mapi.Start()
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
