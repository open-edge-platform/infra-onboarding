// SPDX-FileCopyrightText: 2025 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package worker

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/remotes/docker"
	volumemounts "github.com/docker/docker/volume/mounts"
	"github.com/go-logr/logr"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/tinkerbell/tink/internal/proto"
)

var (
	_ ContainerManager = (*containerdManager)(nil)
	_ LogCapturer      = (*containerdLogCapturer)(nil)

	mountExcluded = []string{
		"/mnt",
		"/sys",
		"/dev/console",
		"/dev",
		"/worker",
		"/lib/modules",
		"/lib/firmware",
		"/workflow",
		"/etc/hosts",
		"/etc/resolv.conf",
		"/etc/localtime",
	}
	parser  = volumemounts.NewLinuxParser()
	randGen = rand.New(rand.NewSource(time.Now().UnixNano()))
)

const (
	namespace   = "tinkerbell"
	socketPath  = "/run/containerd/containerd.sock"
	letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

type containerdManager struct {
	logger          logr.Logger
	registryDetails RegistryConnDetails
	namespace       string
	client          *containerd.Client
	socketPath      string
}

func NewContainerdManager(logger logr.Logger, registryDetails RegistryConnDetails) ContainerManager {
	client, err := containerd.New(socketPath, containerd.WithDefaultNamespace(namespace))
	if err != nil {
		panic(fmt.Errorf("error creating containerd client: %w", err))
	}
	return &containerdManager{
		logger:          logger,
		registryDetails: registryDetails,
		namespace:       namespace,
		socketPath:      socketPath,
		client:          client,
	}
}

// CreateContainer implements ContainerManager.
func (c *containerdManager) CreateContainer(ctx context.Context, cmd []string, wfID string, action *proto.WorkflowAction,
	_ bool, _ bool,
) (string, error) {
	l := c.logger.WithValues("action", action.GetName(), "workflowID", wfID)
	l.Info("creating container", "command", cmd)

	// set up a containerd namespace
	ctx = namespaces.WithNamespace(ctx, c.namespace)

	imageName := action.GetImage()
	image, err := c.pullImageByName(ctx, imageName)
	if err != nil {
		return "", err
	}

	// Prepare workflow directory and mounts
	wfDir := filepath.Join(defaultDataDir, wfID)
	if err := EnsureFolder(wfDir); err != nil {
		return "", err
	}

	mounts := []specs.Mount{
		{
			Destination: "/sys",
			Source:      "/sys",
			Type:        "sysfs",
			Options:     []string{"rbind", "rw"},
		},
		{
			Source:      "/dev",
			Destination: "/dev",
			Type:        "bind",
			Options:     []string{"rbind", "rw"},
		},
		{
			Source:      "/mnt",
			Destination: "/mnt",
			Type:        "bind",
			Options:     []string{"rbind", "rw"},
		},
		{
			Source:      "/dev/console",
			Destination: "/dev/console",
			Type:        "bind",
			Options:     []string{"rbind", "rw"},
		},
		{
			Source:      "/lib/modules",
			Destination: "/lib/modules",
			Type:        "bind",
			Options:     []string{"rbind", "ro"},
		},
		{
			Source:      "/lib/firmware",
			Destination: "/lib/firmware",
			Type:        "bind",
			Options:     []string{"rbind", "rw"},
		},
		{
			Source:      "/worker",
			Destination: "/worker",
			Type:        "bind",
			Options:     []string{"rbind", "rw"},
		},
		{
			Source:      wfDir,
			Destination: "/workflow",
			Type:        "bind",
			Options:     []string{"rbind", "rw"},
		},
	}

	// Add additional volumes from the action
	avs, err := parseVolumes(action.GetVolumes())
	if err != nil {
		return "", errors.Wrap(err, "failed to parse volumes")
	}
	for _, mount := range avs {
		if isValidDst(mount.Destination) {
			mounts = append(mounts, mount)
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		l.Error(err, "failed to get hostname")
	}
	// Create the container specification
	opts := []oci.SpecOpts{
		oci.WithDefaultSpec(),
		oci.WithDefaultUnixDevices,
		oci.WithImageConfig(image),
		oci.WithEnv(action.GetEnvironment()),
		oci.WithMounts(mounts),
		oci.WithCapabilities([]string{"CAP_SYS_ADMIN"}),
		oci.WithHostNamespace(specs.NetworkNamespace),
		oci.WithHostHostsFile,
		oci.WithHostResolvconf,
		// oci.WithHostLocaltime,
		oci.WithEnv([]string{fmt.Sprintf("HOSTNAME=%s", hostname)}),
		oci.WithPrivileged, oci.WithAllDevicesAllowed, oci.WithHostDevices,
	}

	if len(cmd) > 0 {
		opts = append(opts, oci.WithProcessArgs(cmd...))
	}

	if pidConfig := action.GetPid(); pidConfig != "" {
		opts = append(opts, oci.WithHostNamespace(specs.PIDNamespace))
	}

	name := newContainerName(action.GetName())
	snps := c.client.SnapshotService(containerd.DefaultSnapshotter)
	if _, err := snps.Stat(ctx, name); err == nil {
		l.Info("snapshot exists, removing snapshot", "snapshot", name)
		if err := snps.Remove(ctx, name); err != nil {
			l.Error(err, "failed to delete snapshot", "snapshot", name)
		}
	}
	container, err := c.client.NewContainer(
		ctx,
		name,
		containerd.WithSnapshotter(containerd.DefaultSnapshotter),
		containerd.WithNewSnapshot(name, image),
		containerd.WithNewSpec(opts...),
		containerd.WithImage(image),
	)
	if err != nil {
		return "", errors.Wrap(err, "CONTAINERD CREATE")
	}

	return container.ID(), nil
}

// PullImage implements ContainerManager.
func (c *containerdManager) PullImage(ctx context.Context, imageName string) error {
	// set up a containerd namespace
	ctx = namespaces.WithNamespace(ctx, c.namespace)
	_, err := c.pullImageByName(ctx, imageName)
	return err
}

func (c *containerdManager) pullImageByName(ctx context.Context, imageName string) (containerd.Image, error) {
	l := c.logger.WithValues("image", imageName)
	l.Info("pulling image")

	image, err := c.client.GetImage(ctx, imageName)
	if err != nil {
		opts := []containerd.RemoteOpt{containerd.WithPullUnpack}
		if c.registryDetails.Registry != "" {
			// Create a resolver with authentication details
			resolver := docker.NewResolver(docker.ResolverOptions{
				Hosts: func(_ string) ([]docker.RegistryHost, error) {
					return []docker.RegistryHost{
						{
							Host: c.registryDetails.Registry,
							Authorizer: docker.NewDockerAuthorizer(docker.WithAuthCreds(func(_ string) (string, string, error) {
								return c.registryDetails.Username, c.registryDetails.Password, nil
							})),
							Capabilities: docker.HostCapabilityPull,
						},
					}, nil
				},
			})
			opts = append(opts, containerd.WithResolver(resolver))
		}
		// if the image is not in namespaced context, then pull it
		image, err = c.client.Pull(ctx, imageName, opts...)
		if err != nil {
			return image, fmt.Errorf("error pulling image: %w", err)
		}
	}

	l.Info("image pulled", "image", image.Name())
	return image, nil
}

// RemoveContainer implements ContainerManager.
func (c *containerdManager) RemoveContainer(ctx context.Context, id string) error {
	l := c.logger.WithValues("containerID", id)
	l.Info("removing container")
	// set up a containerd namespace
	ctx = namespaces.WithNamespace(ctx, c.namespace)

	container, err := c.client.LoadContainer(ctx, id)
	if err != nil {
		return errors.Wrap(err, "failed to load container")
	}

	task, err := container.Task(ctx, nil)
	if err == nil { // Task exists
		status, err := task.Status(ctx)
		if err != nil {
			return fmt.Errorf("failed to get task status: %w", err)
		}

		if status.Status == containerd.Running {
			if err := task.Kill(ctx, syscall.SIGKILL); err != nil {
				return fmt.Errorf("failed to kill task: %w", err)
			}
		}

		_, err = task.Delete(ctx)
		if err != nil {
			return fmt.Errorf("failed to delete task: %w", err)
		}
	}

	// delete the container
	err = container.Delete(ctx, containerd.WithSnapshotCleanup)
	if err != nil {
		return errors.Wrap(err, "CONTAINERD REMOVE")
	}

	return nil
}

// StartContainer implements ContainerManager.
func (c *containerdManager) StartContainer(ctx context.Context, id string) error {
	l := c.logger.WithValues("containerID", id)
	l.Info("starting container")
	// set up a containerd namespace
	ctx = namespaces.WithNamespace(ctx, c.namespace)

	container, err := c.client.LoadContainer(ctx, id)
	if err != nil {
		return errors.Wrap(err, "CONTAINERD LOAD")
	}

	// Create the task
	task, err := container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
	if err != nil {
		return errors.Wrap(err, "CONTAINERD TASK CREATE")
	}

	// resources := &specs.LinuxResources{
	// 	Devices: []specs.LinuxDeviceCgroup{
	// 		{
	// 			Allow:  true,  // Allow access to devices
	// 			Access: "rwm", // Read, write, and mknod permissions
	// 			Type:   "a",   // Apply to all device types
	// 			Major:  nil,   // Wildcard for all major numbers
	// 			Minor:  nil,   // Wildcard for all minor numbers
	// 		},
	// 	},
	// }
	// if err := task.Update(ctx, containerd.WithResources(resources)); err != nil {
	// 	return errors.Wrap(err, "CONTAINERD TASK UPDATE")
	// }

	// Start the task
	if err := task.Start(ctx); err != nil {
		_, _ = task.Delete(ctx)
		return errors.Wrap(err, "CONTAINERD TASK START")
	}

	return nil
}

// WaitForContainer implements ContainerManager.
func (c *containerdManager) WaitForContainer(ctx context.Context, id string) (proto.State, error) {
	l := c.logger.WithValues("containerID", id)
	l.Info("waiting container")
	// set up a containerd namespace
	ctx = namespaces.WithNamespace(ctx, c.namespace)

	container, err := c.client.LoadContainer(ctx, id)
	if err != nil {
		return proto.State_STATE_FAILED, err
	}

	// get the task associated with the container
	task, err := container.Task(ctx, nil)
	if err != nil {
		return proto.State_STATE_FAILED, err
	}

	var exitStatusC <-chan containerd.ExitStatus
	exitStatusC, err = task.Wait(ctx)
	if err != nil {
		return proto.State_STATE_FAILED, fmt.Errorf("error waiting on task: %w", err)
	}

	select {
	case exitStatus := <-exitStatusC:
		if exitStatus.ExitCode() == 0 {
			return proto.State_STATE_SUCCESS, nil
		}
		return proto.State_STATE_FAILED, nil
	case <-ctx.Done():
		return proto.State_STATE_TIMEOUT, ctx.Err()
	}
}

// WaitForFailedContainer implements ContainerManager.
func (c *containerdManager) WaitForFailedContainer(ctx context.Context, id string, failedActionStatus chan proto.State) {
	l := c.logger.WithValues("containerID", id)
	l.Info("waiting failed container")
	// set up a containerd namespace
	ctx = namespaces.WithNamespace(ctx, c.namespace)

	container, err := c.client.LoadContainer(ctx, id)
	if err != nil {
		failedActionStatus <- proto.State_STATE_FAILED
		return
	}

	// get the task associated with the container
	task, err := container.Task(ctx, nil)
	if err != nil {
		l.Error(err, "error loading task")
		failedActionStatus <- proto.State_STATE_FAILED
		return
	}

	var exitStatusC <-chan containerd.ExitStatus
	exitStatusC, err = task.Wait(ctx)
	if err != nil {
		l.Error(err, "error waiting on task")
		failedActionStatus <- proto.State_STATE_FAILED
		return
	}

	select {
	case exitStatus := <-exitStatusC:
		if exitStatus.ExitCode() == 0 {
			failedActionStatus <- proto.State_STATE_SUCCESS
			return
		}
		failedActionStatus <- proto.State_STATE_FAILED
	case <-ctx.Done():
		l.Error(ctx.Err(), "context done")
		failedActionStatus <- proto.State_STATE_TIMEOUT
	}
}

type containerdLogCapturer struct{}

func NewContainerdLogCapturer() LogCapturer {
	return &containerdLogCapturer{}
}

// CaptureLogs streams container logs to the capturer's writer.
func (l *containerdLogCapturer) CaptureLogs(_ context.Context, _ string) {}

func Init() error {
	if err := EnsureFolder("/worker"); err != nil {
		return err
	}
	if err := EnsureFolder("/lib/firmware"); err != nil {
		return err
	}
	content, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return err
	}
	cmdLines := strings.Split(string(content), " ")
	cfg := parseCmdLine(cmdLines)
	envs := []string{
		fmt.Sprintf("DOCKER_REGISTRY=%s", cfg.registry),
		fmt.Sprintf("REGISTRY_USERNAME=%s", cfg.username),
		fmt.Sprintf("REGISTRY_PASSWORD=%s", cfg.password),
		fmt.Sprintf("TINKERBELL_GRPC_AUTHORITY=%s", cfg.grpcAuthority),
		fmt.Sprintf("TINKERBELL_TLS=%s", cfg.tinkServerTLS),
		fmt.Sprintf("TINKERBELL_INSECURE_TLS=%s", cfg.tinkServerInsecureTLS),
		fmt.Sprintf("WORKER_ID=%s", cfg.workerID),
		fmt.Sprintf("ID=%s", cfg.workerID),
		fmt.Sprintf("HTTP_PROXY=%s", cfg.httpProxy),
		fmt.Sprintf("HTTPS_PROXY=%s", cfg.httpsProxy),
		fmt.Sprintf("NO_PROXY=%s", cfg.noProxy),
	}

	for _, env := range envs {
		kv := splitEnv(env)
		if err := os.Setenv(kv[0], kv[1]); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", kv[0], err)
		}
	}

	go rebootWatch()
	return nil
}

func EnsureFolder(folder string) error {
	// Check if the folder exists
	info, err := os.Stat(folder)
	if os.IsNotExist(err) { //nolint:gocritic // ignore ifElseChain
		// Folder does not exist, create it with 755 permissions
		if err := os.MkdirAll(folder, 0o755); err != nil {
			return fmt.Errorf("failed to create %s folder: %w", folder, err)
		}
		fmt.Printf("%s folder created with 755 permissions", folder)
	} else if err != nil {
		// Other errors (permission issues)
		return fmt.Errorf("error checking %s folder: %w", folder, err)
	} else if !info.IsDir() {
		// Path exists but is not a directory
		return fmt.Errorf("%s exists but is not a directory", folder)
	} else {
		fmt.Printf("%s folder already exists", folder)
	}

	return nil
}

func rebootWatch() {
	fmt.Println("Starting Reboot Watcher")

	// Forever loop
	for {
		if fileExists("/worker/reboot") {
			cmd := exec.Command("/sbin/reboot")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				fmt.Printf("error calling /sbin/reboot: %v\n", err)
				time.Sleep(time.Second)
				continue
			}
			break
		}
		// Wait one second before looking for file
		time.Sleep(time.Second)
	}
	fmt.Println("Rebooting")
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

type tinkWorkerConfig struct {
	// Registry configuration
	registry string
	username string
	password string

	// Tink Server GRPC address:port
	grpcAuthority string

	// Worker ID
	workerID string

	// tinkWorkerImage is the Tink worker image location.
	tinkWorkerImage string

	// tinkServerTLS is whether or not to use TLS for tink-server communication.
	tinkServerTLS string

	// tinkServerInsecureTLS is whether or not to use insecure TLS for tink-server communication; only applies is TLS itself is on
	tinkServerInsecureTLS string

	httpProxy  string
	httpsProxy string
	noProxy    string
}

func parseCmdLine(cmdLines []string) (cfg tinkWorkerConfig) {
	for i := range cmdLines {
		cmdLine := strings.SplitN(cmdLines[i], "=", 2)
		if len(cmdLine) < 2 {
			continue
		}

		switch cmd := cmdLine[0]; cmd {
		case "docker_registry":
			cfg.registry = cmdLine[1]
		case "registry_username":
			cfg.username = cmdLine[1]
		case "registry_password":
			cfg.password = cmdLine[1]
		case "grpc_authority":
			cfg.grpcAuthority = cmdLine[1]
		case "worker_id":
			cfg.workerID = cmdLine[1]
		case "tink_worker_image":
			cfg.tinkWorkerImage = cmdLine[1]
		case "tinkerbell_tls":
			cfg.tinkServerTLS = cmdLine[1]
		case "tinkerbell_insecure_tls":
			cfg.tinkServerInsecureTLS = cmdLine[1]
		case "HTTP_PROXY":
			cfg.httpProxy = cmdLine[1]
		case "HTTPS_PROXY":
			cfg.httpsProxy = cmdLine[1]
		case "NO_PROXY":
			cfg.noProxy = cmdLine[1]
		}
	}
	return cfg
}

func splitEnv(env string) []string {
	kv := strings.SplitN(env, "=", 2)
	if len(kv) == 2 {
		return kv
	}
	return nil
}

func isValidDst(dst string) bool {
	return !slices.Contains(mountExcluded, dst)
}

func parseVolumes(volumes []string) ([]specs.Mount, error) {
	var mounts []specs.Mount
	for _, volume := range volumes {
		mp, err := parser.ParseMountRaw(volume, "")
		if err != nil {
			return nil, fmt.Errorf("failed to parse volume %s: %w", volume, err)
		}
		m := specs.Mount{
			Source:      mp.Spec.Source,
			Destination: mp.Spec.Target,
			Type:        string(mp.Spec.Type),
			Options:     []string{"rbind"},
		}
		if mp.Spec.ReadOnly {
			m.Options = append(m.Options, "ro")
		} else {
			m.Options = append(m.Options, "rw")
		}
		mounts = append(mounts, m)
	}
	return mounts, nil
}

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

func randStr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[randGen.Intn(len(letterBytes))]
	}
	return string(b)
}

func newContainerName(name string) string {
	// max length is 76 in containerd.
	return fmt.Sprintf("%s-%s", truncateStr(name, 60), randStr(10))
}
