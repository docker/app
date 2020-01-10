package docker

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	unix_path "path"

	"github.com/cnabio/cnab-go/driver"
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
func (d *Driver) Run(op *driver.Operation) (driver.OperationResult, error) {
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
		"OUTPUTS_MOUNT_PATH":  "Absolute path to where Docker driver can create temporary directories to bundle outputs. Defaults to temp dir.",
		"CLEANUP_CONTAINERS":  "If true, the docker container will be destroyed when it finishes running. If false, it will not be destroyed. The supported values are true and false. Defaults to true.",
	}
}

// SetConfig sets Docker driver configuration
func (d *Driver) SetConfig(settings map[string]string) {
	// Set default and provide feedback on acceptable input values.
	value, ok := settings["CLEANUP_CONTAINERS"]
	if !ok {
		settings["CLEANUP_CONTAINERS"] = "true"
	} else if value != "true" && value != "false" {
		fmt.Printf("CLEANUP_CONTAINERS environment variable has unexpected value %q. Supported values are 'true', 'false', or unset.", value)
	}

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

func (d *Driver) exec(op *driver.Operation) (driver.OperationResult, error) {
	ctx := context.Background()

	cli, err := d.initializeDockerCli()
	if err != nil {
		return driver.OperationResult{}, err
	}

	if d.Simulate {
		return driver.OperationResult{}, nil
	}
	if d.config["PULL_ALWAYS"] == "1" {
		if err := pullImage(ctx, cli, op.Image.Image); err != nil {
			return driver.OperationResult{}, err
		}
	}
	var env []string
	for k, v := range op.Environment {
		env = append(env, fmt.Sprintf("%s=%v", k, v))
	}

	cfg := &container.Config{
		Image:        op.Image.Image,
		Env:          env,
		Entrypoint:   strslice.StrSlice{"/cnab/app/run"},
		AttachStderr: true,
		AttachStdout: true,
	}

	hostCfg := &container.HostConfig{}
	for _, opt := range d.dockerConfigurationOptions {
		if err := opt(cfg, hostCfg); err != nil {
			return driver.OperationResult{}, err
		}
	}

	resp, err := cli.Client().ContainerCreate(ctx, cfg, hostCfg, nil, "")
	switch {
	case client.IsErrNotFound(err):
		fmt.Fprintf(cli.Err(), "Unable to find image '%s' locally\n", op.Image.Image)
		if err := pullImage(ctx, cli, op.Image.Image); err != nil {
			return driver.OperationResult{}, err
		}
		if resp, err = cli.Client().ContainerCreate(ctx, cfg, hostCfg, nil, ""); err != nil {
			return driver.OperationResult{}, fmt.Errorf("cannot create container: %v", err)
		}
	case err != nil:
		return driver.OperationResult{}, fmt.Errorf("cannot create container: %v", err)
	}

	if d.config["CLEANUP_CONTAINERS"] == "true" {
		defer cli.Client().ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{})
	}

	tarContent, err := generateTar(op.Files)
	if err != nil {
		return driver.OperationResult{}, fmt.Errorf("error staging files: %s", err)
	}
	options := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
	}
	// This copies the tar to the root of the container. The tar has been assembled using the
	// path from the given file, starting at the /.
	err = cli.Client().CopyToContainer(ctx, resp.ID, "/", tarContent, options)
	if err != nil {
		return driver.OperationResult{}, fmt.Errorf("error copying to / in container: %s", err)
	}

	attach, err := cli.Client().ContainerAttach(ctx, resp.ID, types.ContainerAttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
		Logs:   true,
	})
	if err != nil {
		return driver.OperationResult{}, fmt.Errorf("unable to retrieve logs: %v", err)
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

	if err = cli.Client().ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return driver.OperationResult{}, fmt.Errorf("cannot start container: %v", err)
	}
	statusc, errc := cli.Client().ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errc:
		if err != nil {
			opResult, fetchErr := d.fetchOutputs(ctx, resp.ID, op)
			return opResult, containerError("error in container", err, fetchErr)
		}
	case s := <-statusc:
		if s.StatusCode == 0 {
			return d.fetchOutputs(ctx, resp.ID, op)
		}
		if s.Error != nil {
			opResult, fetchErr := d.fetchOutputs(ctx, resp.ID, op)
			return opResult, containerError(fmt.Sprintf("container exit code: %d, message", s.StatusCode), err, fetchErr)
		}
		opResult, fetchErr := d.fetchOutputs(ctx, resp.ID, op)
		return opResult, containerError(fmt.Sprintf("container exit code: %d, message", s.StatusCode), err, fetchErr)
	}
	opResult, fetchErr := d.fetchOutputs(ctx, resp.ID, op)
	if fetchErr != nil {
		return opResult, fmt.Errorf("fetching outputs failed: %s", fetchErr)
	}
	return opResult, err
}

func containerError(containerMessage string, containerErr, fetchErr error) error {
	if fetchErr != nil {
		return fmt.Errorf("%s: %v. fetching outputs failed: %s", containerMessage, containerErr, fetchErr)
	}

	return fmt.Errorf("%s: %v", containerMessage, containerErr)
}

// fetchOutputs takes a context and a container ID; it copies the /cnab/app/outputs directory from that container.
// The goal is to collect all the files in the directory (recursively) and put them in a flat map of path to contents.
// This map will be inside the OperationResult. When fetchOutputs returns an error, it may also return partial results.
func (d *Driver) fetchOutputs(ctx context.Context, container string, op *driver.Operation) (driver.OperationResult, error) {
	opResult := driver.OperationResult{
		Outputs: map[string]string{},
	}
	// The /cnab/app/outputs directory probably only exists if outputs are created. In the
	// case there are no outputs defined on the operation, there probably are none to copy
	// and we should return early.
	if len(op.Outputs) == 0 {
		return opResult, nil
	}
	ioReader, _, err := d.dockerCli.Client().CopyFromContainer(ctx, container, "/cnab/app/outputs")
	if err != nil {
		return opResult, fmt.Errorf("error copying outputs from container: %s", err)
	}

	tarReader := tar.NewReader(ioReader)
	header, err := tarReader.Next()

	// io.EOF pops us out of loop on successful run.
	for err == nil {
		// skip directories because we're gathering file contents
		if header.FileInfo().IsDir() {
			header, err = tarReader.Next()
			continue
		}

		var contents []byte
		// CopyFromContainer strips prefix above outputs directory.
		pathInContainer := unix_path.Join("/cnab", "app", header.Name)

		contents, err = ioutil.ReadAll(tarReader)
		if err != nil {
			return opResult, fmt.Errorf("error while reading %q from outputs tar: %s", pathInContainer, err)
		}
		opResult.Outputs[pathInContainer] = string(contents)
		header, err = tarReader.Next()
	}

	if err != io.EOF {
		return opResult, err
	}

	// if an applicable output is expected but does not exist and it has a
	// non-empty default value, create an entry in the map with the
	// default value as its contents
	for name, output := range op.Bundle.Outputs {
		filepath := unix_path.Join("/cnab", "app", "outputs", name)
		if !existsInOutputsMap(opResult.Outputs, filepath) && output.AppliesTo(op.Action) {
			if outputDefinition, exists := op.Bundle.Definitions[output.Definition]; exists {
				outputDefault := outputDefinition.Default
				if outputDefault != nil {
					contents := fmt.Sprintf("%v", outputDefault)
					opResult.Outputs[filepath] = contents
				} else {
					return opResult, fmt.Errorf("required output %s is missing and has no default", name)
				}
			}
		}
	}

	return opResult, nil
}

func existsInOutputsMap(outputsMap map[string]string, path string) bool {
	for outputPath := range outputsMap {
		if outputPath == path {
			return true
		}
	}
	return false
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
