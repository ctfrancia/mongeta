package config

import "time"

type Config struct {
	Worker  WorkerConfig
	Manager ManagerConfig
	Server  ServerConfig
}

type WorkerConfig struct {
	Host           string        `env:"MONGETA_WORKER_HOST" envDefault:"localhost"`
	Port           int           `env:"MONGETA_WORKER_PORT" envDefault:"8080"`
	QueueSize      int           `env:"MONGETA_WORKER_QUEUE_SIZE" envDefault:"100"`
	RunInterval    time.Duration `env:"MONGETA_WORKER_RUN_INTERVAL" envDefault:"10s"`
	StatsInterval  time.Duration `env:"MONGETA_WORKER_STATS_INTERVAL" envDefault:"15s"`
	UpdateInterval time.Duration `env:"MONGETA_WORKER_UPDATE_INTERVAL" envDefault:"15s"`
}

type ManagerConfig struct {
	Host                string        `env:"MONGETA_HOST" envDefault:"localhost"`
	Port                int           `env:"MONGETA_PORT" envDefault:"8081"`
	QueueSize           int           `env:"MONGETA_MANAGER_QUEUE_SIZE" envDefault:"100"`
	ProcessInterval     time.Duration `env:"MONGETA_MANAGER_PROCESS_INTERVAL" envDefault:"10s"`
	UpdateInterval      time.Duration `env:"MONGETA_MANAGER_UPDATE_INTERVAL" envDefault:"15s"`
	MaxRestarts         int           `env:"MONGETA_MANAGER_MAX_RESTARTS" envDefault:"3"`
	HealthCheckInterval time.Duration `env:"MONGETA_MANAGER_HEALTH_INTERVAL" envDefault:"20s"`
}

type ServerConfig struct {
	ReadTimeout  time.Duration `env:"MONGETA_SERVER_READ_TIMEOUT"  envDefault:"5s"`
	WriteTimeout time.Duration `env:"MONGETA_SERVER_WRITE_TIMEOUT" envDefault:"10s"`
	IdleTimeout  time.Duration `env:"MONGETA_SERVER_IDLE_TIMEOUT" envDefault:"120s"`
}
