// Package task handles the lowest form of an action
package task

import (
	"context"
	"io"
	"math"
	"os"
	"time"

	"github.com/ctfrancia/mongeta/logger"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"
)

type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

type Task struct {
	ID            uuid.UUID
	ContainerID   string
	Name          string
	State         State
	Image         string
	CPU           float64
	Memory        int64
	Disk          int64
	ExposedPorts  nat.PortSet
	HostPorts     nat.PortMap
	PortBindings  map[string]string
	RestartPolicy container.RestartPolicyMode
	StartTime     time.Time
	FinishTime    time.Time
	HealthCheck   string
	RestartCount  int
}

type TaskEvent struct {
	ID        uuid.UUID
	State     State
	TimeStamp time.Time
	Task      Task
}

type Config struct {
	Name          string
	AttachStdin   bool
	AttachStdout  bool
	AttachStderr  bool
	ExposedPorts  nat.PortSet
	Cmd           []string
	Image         string
	CPU           float64
	Memory        int64
	Disk          int64
	Env           []string
	RestartPolicy container.RestartPolicyMode
}

type Docker struct {
	Client *client.Client
	Config Config
}

type DockerInspectResponse struct {
	Error     error
	Container *container.InspectResponse
}

type DockerResult struct {
	Error       error
	Action      string
	ContainerID string
	Result      string
}

func NewConfig(t *Task) *Config {
	return &Config{
		Name:          t.Name,
		ExposedPorts:  t.ExposedPorts,
		Image:         t.Image,
		CPU:           t.CPU,
		Memory:        t.Memory,
		Disk:          t.Disk,
		RestartPolicy: t.RestartPolicy,
	}
}

func NewDocker(c *Config) (*Docker, error) {
	dc, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, err
	}

	return &Docker{
		Client: dc,
		Config: *c,
	}, nil
}

func (d *Docker) Run() DockerResult {
	ctx := context.Background()
	reader, err := d.Client.ImagePull(ctx, d.Config.Image, image.PullOptions{})
	if err != nil {
		logger.Error("error pulling image", "image", d.Config.Image, "err", err)
		return DockerResult{Error: err}
	}
	io.Copy(os.Stdout, reader)

	rp := container.RestartPolicy{
		Name: d.Config.RestartPolicy,
	}

	r := container.Resources{
		Memory:   d.Config.Memory,
		NanoCPUs: int64(d.Config.CPU * math.Pow(10, 9)),
	}

	cc := container.Config{
		Image:        d.Config.Image,
		Tty:          false,
		Env:          d.Config.Env,
		ExposedPorts: d.Config.ExposedPorts,
		AttachStdin:  d.Config.AttachStdin,
		AttachStdout: d.Config.AttachStdout,
		AttachStderr: d.Config.AttachStderr,
	}

	hc := container.HostConfig{
		RestartPolicy:   rp,
		Resources:       r,
		PublishAllPorts: true,
	}
	resp, err := d.Client.ContainerCreate(ctx, &cc, &hc, nil, nil, d.Config.Name)
	if err != nil {
		logger.Error("error creating container", "name", d.Config.Name, "err", err)
		return DockerResult{Error: err, Action: "Create", Result: d.Config.Name}
	}

	err = d.Client.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		logger.Error("error starting container", "name", d.Config.Name, "err", err)
		return DockerResult{Error: err, Action: "Start", Result: d.Config.Name}
	}

	out, err := d.Client.ContainerLogs(
		ctx,
		resp.ID,
		container.LogsOptions{ShowStdout: true, ShowStderr: true},
	)
	if err != nil {
		logger.Error("error getting container logs", "name", d.Config.Name, "err", err)
		return DockerResult{Error: err, Action: "Logs", Result: d.Config.Name}
	}
	stdcopy.StdCopy(os.Stdout, os.Stderr, out)
	return DockerResult{ContainerID: resp.ID, Action: "Start", Result: "Success"}
}

func (d *Docker) Stop(ID string) DockerResult {
	logger.Info("stopping container", "container_id", ID)
	ctx := context.Background()
	err := d.Client.ContainerStop(ctx, ID, container.StopOptions{})
	if err != nil {
		logger.Error("error stopping container", "container_id", ID, "err", err)
		return DockerResult{Error: err, Action: "Stop", Result: ID}
	}

	err = d.Client.ContainerRemove(ctx, ID, container.RemoveOptions{})
	if err != nil {
		logger.Error("error removing container", "container_id", ID, "err", err)
		return DockerResult{Error: err, Action: "Remove", Result: ID}
	}

	return DockerResult{Action: "Stop", Result: "Success"}
}

func (d *Docker) Inspect(ctx context.Context, containerID string) DockerInspectResponse {
	resp, err := d.Client.ContainerInspect(ctx, containerID)
	if err != nil {
		logger.Error("error inspecting container", "container_id", containerID, "err", err)
		return DockerInspectResponse{Error: err}
	}

	return DockerInspectResponse{Container: &resp}
}
