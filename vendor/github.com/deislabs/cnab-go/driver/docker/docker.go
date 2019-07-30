package docker

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	unix_path "path"

	"github.com/deislabs/cnab-go/driver"
	"github.com/docker/cli/cli/command"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/docker/registry"
)

// Driver is capable of running Docker invocation images using Docker itself.
type Driver struct {
	config map[string]string
	// If true, this will not actually run Docker
	Simulate                   bool
	dockerCli                  command.Cli
	dockerConfigurationOptions []ConfigurationOption
	containerOut               io.Writer
	containerErr               io.Writer
}

// Run executes the Docker driver
func (d *Driver) Run(op *driver.Operation) error {
	return d.exec(op)
}

// Handles indicates that the Docker driver supports "docker" and "oci"
func (d *Driver) Handles(dt string) bool {
	return dt == driver.ImageTypeDocker || dt == driver.ImageTypeOCI
}

// AddConfigurationOptions adds configuration callbacks to the driver
func (d *Driver) AddConfigurationOptions(opts ...ConfigurationOption) {
	d.dockerConfigurationOptions = append(d.dockerConfigurationOptions, opts...)
}

// Config returns the Docker driver configuration options
func (d *Driver) Config() map[string]string {
	return map[string]string{
		"VERBOSE":             "Increase verbosity. true, false are supported values",
		"PULL_ALWAYS":         "Always pull image, even if locally available (0|1)",
		"DOCKER_DRIVER_QUIET": "Make the Docker driver quiet (only print container stdout/stderr)",
	}
}

// SetConfig sets Docker driver configuration
func (d *Driver) SetConfig(settings map[string]string) {
	d.config = settings
}

// SetDockerCli makes the driver use an already initialized cli
func (d *Driver) SetDockerCli(dockerCli command.Cli) {
	d.dockerCli = dockerCli
}

// SetContainerOut sets the container output stream
func (d *Driver) SetContainerOut(w io.Writer) {
	d.containerOut = w
}

// SetContainerErr sets the container error stream
func (d *Driver) SetContainerErr(w io.Writer) {
	d.containerErr = w
}

func pullImage(ctx context.Context, cli command.Cli, image string) error {
	ref, err := reference.ParseNormalizedNamed(image)
	if err != nil {
		return err
	}

	// Resolve the Repository name from fqn to RepositoryInfo
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return err
	}
	authConfig := command.ResolveAuthConfig(ctx, cli, repoInfo.Index)
	encodedAuth, err := command.EncodeAuthToBase64(authConfig)
	if err != nil {
		return err
	}
	options := types.ImagePullOptions{
		RegistryAuth: encodedAuth,
	}
	responseBody, err := cli.Client().ImagePull(ctx, image, options)
	if err != nil {
		return err
	}
	defer responseBody.Close()

	// passing isTerm = false here because of https://github.com/Nvveen/Gotty/pull/1
	return jsonmessage.DisplayJSONMessagesStream(responseBody, cli.Out(), cli.Out().FD(), false, nil)
}

func (d *Driver) initializeDockerCli() (command.Cli, error) {
	if d.dockerCli != nil {
		return d.dockerCli, nil
	}
	cli, err := command.NewDockerCli()
	if err != nil {
		return nil, err
	}
	if d.config["DOCKER_DRIVER_QUIET"] == "1" {
		cli.Apply(command.WithCombinedStreams(ioutil.Discard))
	}
	if err := cli.Initialize(cliflags.NewClientOptions()); err != nil {
		return nil, err
	}
	d.dockerCli = cli
	return cli, nil
}

func (d *Driver) exec(op *driver.Operation) error {
	ctx := context.Background()

	cli, err := d.initializeDockerCli()
	if err != nil {
		return err
	}

	if d.Simulate {
		return nil
	}
	if d.config["PULL_ALWAYS"] == "1" {
		if err := pullImage(ctx, cli, op.Image); err != nil {
			return err
		}
	}
	var env []string
	for k, v := range op.Environment {
		env = append(env, fmt.Sprintf("%s=%v", k, v))
	}

	cfg := &container.Config{
		Image:        op.Image,
		Env:          env,
		Entrypoint:   strslice.StrSlice{"/cnab/app/run"},
		AttachStderr: true,
		AttachStdout: true,
	}

	hostCfg := &container.HostConfig{AutoRemove: true}

	for _, opt := range d.dockerConfigurationOptions {
		if err := opt(cfg, hostCfg); err != nil {
			return err
		}
	}

	resp, err := cli.Client().ContainerCreate(ctx, cfg, hostCfg, nil, "")
	switch {
	case client.IsErrNotFound(err):
		fmt.Fprintf(cli.Err(), "Unable to find image '%s' locally\n", op.Image)
		if err := pullImage(ctx, cli, op.Image); err != nil {
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
	var (
		stdout io.Writer = os.Stdout
		stderr io.Writer = os.Stderr
	)
	if d.containerOut != nil {
		stdout = d.containerOut
	}
	if d.containerErr != nil {
		stderr = d.containerErr
	}
	go func() {
		defer attach.Close()
		for {
			_, err := stdcopy.StdCopy(stdout, stderr, attach.Reader)
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
			return fmt.Errorf("container exit code: %d, message: %v", s.StatusCode, s.Error.Message)
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

// ConfigurationOption is an option used to customize docker driver container and host config
type ConfigurationOption func(*container.Config, *container.HostConfig) error
