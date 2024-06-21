package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

func deviceDiscovery(cfg tinkConfig) error {
	fmt.Println("Starting Device Discovery")

	// Generate the path to the deviceDiscoveryImage
	var imageName string

	if cfg.deviceDiscoveryImage != "" {
		imageName = cfg.deviceDiscoveryImage
	}
	if imageName == "" {
		return fmt.Errorf("The device-discovery image is not specified in the /proc/cmdline.")
	}

	// Generate the configuration of the container
	tinkContainer := &container.Config{
		Image: imageName,
		Env: []string{
			fmt.Sprintf("DOCKER_REGISTRY=%s", cfg.registry),
			fmt.Sprintf("OBM_ADDRESS=%s", cfg.onboardingManagerAddress),
			fmt.Sprintf("MAC_ID=%s", cfg.workerID),
		},
		AttachStdout: true,
		AttachStderr: true,
	}

	tinkHostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/worker",
				Target: "/worker",
			},
			{
				Type:   mount.TypeBind,
				Source: "/var/run/docker.sock",
				Target: "/var/run/docker.sock",
			},
		},
		NetworkMode: "host",
		Privileged:  true,
	}

	// Give time to Docker to start
	// Alternatively we watch for the socket being created
	time.Sleep(time.Second * 3)
	fmt.Println("Starting Communication with Docker Engine")

	// Create Docker client with API (socket)
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("error creating Docker client: %w", err)
	}

	fmt.Printf("Pulling image [%s]", imageName)

	// Image pull operation
	var out io.ReadCloser
	imagePullOperation := func() error {
		out, err = cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
		if err != nil {
			fmt.Printf("Image pull failure %s, %v\n", imageName, err)
			return err
		}
		return nil
	}
	if err = backoff.Retry(imagePullOperation, backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetryAttempts)); err != nil {
		return fmt.Errorf("error pulling image: %w", err)
	}

	if _, err = io.Copy(os.Stdout, out); err != nil {
		return fmt.Errorf("error copying image pull output: %w", err)
	}

	if err = out.Close(); err != nil {
		fmt.Printf("error closing io.ReadCloser out: %s", err)
	}

	// Create container
	resp, err := cli.ContainerCreate(ctx, tinkContainer, tinkHostConfig, nil, nil, "")
	if err != nil {
		return fmt.Errorf("error creating container: %w", err)
	}

	// Start container
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("error starting container: %w", err)
	}

	fmt.Println("Container started with ID:", resp.ID)

	// Wait for the container to complete
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("error waiting for container: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("container exited with non-zero status: %d", status.StatusCode)
		}
	}

	// Fetch logs
	out, err = cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true, Follow: true})
	if err != nil {
		return fmt.Errorf("error fetching container logs: %w", err)
	}
	defer out.Close()

	if _, err = io.Copy(os.Stdout, out); err != nil {
		return fmt.Errorf("error copying container logs: %w", err)
	}

	fmt.Println("Container completed successfully")

	return nil
}
