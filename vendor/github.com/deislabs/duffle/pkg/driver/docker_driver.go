package driver

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	unix_path "path"

	"github.com/docker/cli/cli/command"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/deislabs/duffle/pkg/image"
)

// DockerDriver is capable of running Docker invocation images using Docker itself.
type DockerDriver struct {
	config map[string]string
	// If true, this will not actually run Docker
	Simulate  bool
	dockerCli command.Cli
}

// Run executes the Docker driver
func (d *DockerDriver) Run(op *Operation) error {
	return d.exec(op)
}

// Handles indicates that the Docker driver supports "docker" and "oci"
func (d *DockerDriver) Handles(dt string) bool {
	return dt == ImageTypeDocker || dt == ImageTypeOCI
}

// Config returns the Docker driver configuration options
func (d *DockerDriver) Config() map[string]string {
	return map[string]string{
		"VERBOSE":             "Increase verbosity. true, false are supported values",
		"PULL_ALWAYS":         "Always pull image, even if locally available (0|1)",
		"DOCKER_DRIVER_QUIET": "Make the Docker driver quiet (only print container stdout/stderr)",
	}
}

// SetDockerCli makes the driver use an already initialized cli
func (d *DockerDriver) SetDockerCli(dockerCli command.Cli) {
	d.dockerCli = dockerCli
}

func (d *DockerDriver) initializeDockerCli() (command.Cli, error) {
	if d.dockerCli != nil {
		return d.dockerCli, nil
	}
	var cliout, clierr io.Writer = os.Stdout, os.Stderr
	if d.config["DOCKER_DRIVER_QUIET"] == "1" {
		cliout = nullWriter{}
		clierr = nullWriter{}
	}
	cli := command.NewDockerCli(os.Stdin, cliout, clierr, false, nil)
	if err := cli.Initialize(cliflags.NewClientOptions()); err != nil {
		return nil, err
	}
	d.dockerCli = cli
	return cli, nil
}

// SetConfig sets Docker driver configuration
func (d *DockerDriver) SetConfig(settings map[string]string) {
	d.config = settings
}

type nullWriter struct{}

func (nullWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (d *DockerDriver) exec(op *Operation) error {
	ctx := context.Background()
	cli, err := d.initializeDockerCli()
	if err != nil {
		return err
	}
	if d.Simulate {
		return nil
	}
	if d.config["PULL_ALWAYS"] == "1" {
		if err := image.PullImage(ctx, cli, op.Image); err != nil {
			return err
		}
	}
	var env []string
	for k, v := range op.Environment {
		env = append(env, fmt.Sprintf("%s=%v", k, v))
	}

	mounts := []mount.Mount{
		{
			Type:   mount.TypeBind,
			Source: "/var/run/docker.sock",
			Target: "/var/run/docker.sock",
		},
	}
	cfg := &container.Config{
		Image:        op.Image,
		Env:          env,
		Entrypoint:   strslice.StrSlice{"/cnab/app/run"},
		AttachStderr: true,
		AttachStdout: true,
	}

	hostCfg := &container.HostConfig{Mounts: mounts, AutoRemove: true}

	resp, err := cli.Client().ContainerCreate(ctx, cfg, hostCfg, nil, "")
	switch {
	case client.IsErrNotFound(err):
		fmt.Fprintf(cli.Err(), "Unable to find image '%s' locally\n", op.Image)
		if err := image.PullImage(ctx, cli, op.Image); err != nil {
			return err
		}
		if resp, err = cli.Client().ContainerCreate(ctx, cfg, hostCfg, nil, ""); err != nil {
			return fmt.Errorf("cannot create container: %v", err)
		}
	case err != nil:
		return fmt.Errorf("cannot create container: %v", err)
	}

	tarContent, err := generateTar(op.Files)
	if err != nil {
		return fmt.Errorf("error staging files: %s", err)
	}
	options := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
	}
	// This copies the tar to the root of the container. The tar has been assembled using the
	// path from the given file, starting at the /.
	err = cli.Client().CopyToContainer(ctx, resp.ID, "/", tarContent, options)
	if err != nil {
		return fmt.Errorf("error copying to / in container: %s", err)
	}

	attach, err := cli.Client().ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
		Logs:   true,
	})
	if err != nil {
		return fmt.Errorf("unable to retrieve logs: %v", err)
	}
	go func() {
		defer attach.Close()
		for {
			_, err := stdcopy.StdCopy(os.Stdout, os.Stderr, attach.Reader)
			if err != nil {
				break
			}
		}
	}()

	statusc, errc := cli.Client().ContainerWait(ctx, resp.ID, container.WaitConditionRemoved)
	if err = cli.Client().ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("cannot start container: %v", err)
	}
	select {
	case err := <-errc:
		if err != nil {
			return fmt.Errorf("error in container: %v", err)
		}
	case s := <-statusc:
		if s.StatusCode == 0 {
			return nil
		}
		if s.Error != nil {
			return fmt.Errorf("error in container: %v", s.Error.Message)
		}
		return fmt.Errorf("container exit code: %d", s.StatusCode)
	}
	return err
}

func generateTar(files map[string]string) (io.Reader, error) {
	r, w := io.Pipe()
	tw := tar.NewWriter(w)
	for path := range files {
		if !unix_path.IsAbs(path) {
			return nil, fmt.Errorf("destination path %s should be an absolute unix path", path)
		}
	}
	go func() {
		for path, content := range files {
			hdr := &tar.Header{
				Name: path,
				Mode: 0644,
				Size: int64(len(content)),
			}
			tw.WriteHeader(hdr)
			tw.Write([]byte(content))
		}
		w.Close()
	}()
	return r, nil
}
