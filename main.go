package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ctfrancia/mongeta/task"
	"github.com/ctfrancia/mongeta/worker"

	"github.com/golang-collections/collections/queue"

	"github.com/google/uuid"
	"github.com/moby/moby/client"
)

func main() {
	host := os.Getenv("MONGETA_HOST")
	port, _ := strconv.Atoi(os.Getenv("MONGETA_PORT"))

	fmt.Println("Starting Mongeta worker")

	w := worker.Worker{
		Queue: *queue.New(),
		DB:    make(map[uuid.UUID]*task.Task),
	}
	api := worker.API{Address: host, Port: port, Worker: &w}

	go runTasks(&w)
	go w.CollectStats()
	api.Start()
	/*
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
	*/
}

func runTasks(w *worker.Worker) {
	for {
		if w.Queue.Len() != 0 {
			result := w.RunTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n", result.Error)
			}
		} else {
			log.Printf("No tasks to process currently.\n")
		}
		log.Println("Sleeping for 10 seconds.")
		time.Sleep(10 * time.Second)
	}
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
