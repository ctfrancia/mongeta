package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ctfrancia/mongeta/config"
	"github.com/ctfrancia/mongeta/manager"
	"github.com/ctfrancia/mongeta/task"
	"github.com/ctfrancia/mongeta/worker"

	"github.com/caarlos0/env/v11"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

func main() {
	cfg := config.Config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt,
		syscall.SIGTERM)
	defer stop()

	log.Println("Starting Mongeta")

	w := worker.Worker{
		Queue: *queue.New(),
		DB:    make(map[uuid.UUID]*task.Task),
	}
	wapi := worker.API{Address: cfg.Worker.Host, Port: cfg.Worker.Port, Worker: &w}

	workers := []string{fmt.Sprintf("%s:%d", cfg.Worker.Host, cfg.Worker.Port)}
	m := manager.New(workers)
	mapi := manager.API{Address: cfg.Manager.Host, Port: cfg.Manager.Port, Manager: m}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() { defer wg.Done(); w.RunTasks(ctx, cfg.Worker.RunInterval) }()

	wg.Add(1)
	go func() { defer wg.Done(); w.CollectStats(ctx, cfg.Worker.StatsInterval) }()

	wg.Add(1)
	go func() { defer wg.Done(); wapi.Start(ctx) }()

	wg.Add(1)
	go func() { defer wg.Done(); m.ProcessTasks(ctx, cfg.Manager.ProcessInterval) }()

	wg.Add(1)
	go func() { defer wg.Done(); m.UpdateTasks(ctx, cfg.Manager.UpdateInterval) }()

	wg.Add(1)
	go func() { defer wg.Done(); m.DoHealthChecks(ctx, cfg.Manager.HealthCheckInterval) }()

	wg.Add(1)
	go func() { defer wg.Done(); mapi.Start(ctx) }()

	<-ctx.Done()
	log.Println("Shutdown signal received, waiting for goroutines...")
	wg.Wait()
	log.Println("Clean shutdown complete")
}
