package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/caarlos0/env/v11"
	"github.com/ctfrancia/mongeta/config"
	"github.com/ctfrancia/mongeta/logger"
	"github.com/ctfrancia/mongeta/manager"
	"github.com/ctfrancia/mongeta/worker"
)

func main() {
	logger.Init(logger.Options{
		Level:  logger.LevelInfo,
		Format: logger.FormatText,
	})

	cfg := config.Config{}
	if err := env.Parse(&cfg); err != nil {
		logger.Error("failed to parse config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt,
		syscall.SIGTERM)
	defer stop()

	logger.Info("starting Mongeta")

	w := worker.NewWorker(cfg.Worker.QueueSize)
	wapi := worker.API{
		Address:      cfg.Worker.Host,
		Port:         cfg.Worker.Port,
		Worker:       w,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	workers := []string{fmt.Sprintf("%s:%d", cfg.Worker.Host, cfg.Worker.Port)}
	m := manager.New(workers, cfg.Manager.QueueSize)
	mapi := manager.API{
		Address:      cfg.Manager.Host,
		Port:         cfg.Manager.Port,
		Manager:      m,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

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
	logger.Info("shutdown signal received, waiting for goroutines")
	wg.Wait()
	logger.Info("clean shutdown complete")
}
