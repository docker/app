package e2e

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/app/internal"
	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/icmd"
)

// readFile returns the content of the file at the designated path normalizing
// line endings by removing any \r.
func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := ioutil.ReadFile(path)
	assert.NilError(t, err, "missing '"+path+"' file")
	return strings.Replace(string(content), "\r", "", -1)
}

func runWithDindSwarmAndRegistry(t *testing.T, todo func(dindSwarmAndRegistryInfo)) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	registryPort := findAvailablePort()
	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()

	cmd.Env = append(cmd.Env, "DOCKER_TARGET_CONTEXT=swarm-target-context")

	// The dind doesn't have the cnab-app-base image so we save it in order to load it later
	saveCmd := icmd.Cmd{Command: dockerCli.Command("save", fmt.Sprintf("docker/cnab-app-base:%s", internal.Version), "-o", tmpDir.Join("cnab-app-base.tar.gz"))}
	icmd.RunCmd(saveCmd).Assert(t, icmd.Success)

	// we have a difficult constraint here:
	// - the registry must be reachable from the client side (for cnab-to-oci, which does not use the docker daemon to access the registry)
	// - the registry must be reachable from the dind daemon on the same address/port
	// Solution found is: fix the port of the registry to be the same internally and externally
	// and run the dind container in the same network namespace: this way 127.0.0.1:<registry-port> both resolves to the registry from the client and from dind

	swarm := NewContainer("docker:18.09-dind", 2375, "--insecure-registry", fmt.Sprintf("127.0.0.1:%d", registryPort))
	swarm.Start(t, "--expose", strconv.FormatInt(int64(registryPort), 10),
		"-p", fmt.Sprintf("%d:%d", registryPort, registryPort),
		"-p", "2375")
	defer swarm.Stop(t)

	registry := NewContainer("registry:2", registryPort)
	registry.StartWithContainerNetwork(t, swarm, "-e", "REGISTRY_VALIDATION_MANIFESTS_URLS_ALLOW=[^http]",
		"-e", fmt.Sprintf("REGISTRY_HTTP_ADDR=0.0.0.0:%d", registryPort))
	defer registry.StopNoFail()

	// We  need two contexts:
	// - one for `docker` so that it connects to the dind swarm created before
	// - the target context for the invocation image to install within the swarm
	cmd.Command = dockerCli.Command("context", "create", "swarm-context", "--docker", fmt.Sprintf(`"host=tcp://%s"`, swarm.GetAddress(t)), "--default-stack-orchestrator", "swarm")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// When creating a context on a Windows host we cannot use
	// the unix socket but it's needed inside the invocation image.
	// The workaround is to create a context with an empty host.
	// This host will default to the unix socket inside the
	// invocation image
	cmd.Command = dockerCli.Command("context", "create", "swarm-target-context", "--docker", "host=", "--default-stack-orchestrator", "swarm")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	// Initialize the swarm
	cmd.Env = append(cmd.Env, "DOCKER_CONTEXT=swarm-context")
	cmd.Command = dockerCli.Command("swarm", "init")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
	// Load the needed base cnab image into the swarm docker engine
	cmd.Command = dockerCli.Command("load", "-i", tmpDir.Join("cnab-app-base.tar.gz"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	info := dindSwarmAndRegistryInfo{
		configuredCmd:   cmd,
		registryAddress: registry.GetAddress(t),
		swarmAddress:    swarm.GetAddress(t),
		stopRegistry:    registry.StopNoFail,
		registryLogs:    registry.Logs(t),
	}
	todo(info)
}

// Container represents a docker container
type Container struct {
	image           string
	privatePort     int
	address         string
	container       string
	parentContainer string
	args            []string
}

// NewContainer creates a new Container
func NewContainer(image string, privatePort int, args ...string) *Container {
	return &Container{
		image:       image,
		privatePort: privatePort,
		args:        args,
	}
}

// Start starts a new docker container on a random port
func (c *Container) Start(t *testing.T, dockerArgs ...string) {
	args := []string{"run", "--rm", "--privileged", "-d", "-P"}
	args = append(args, dockerArgs...)
	args = append(args, c.image)
	args = append(args, c.args...)
	result := icmd.RunCommand(dockerCli.path, args...).Assert(t, icmd.Success)
	c.container = strings.Trim(result.Stdout(), " \r\n")
	time.Sleep(time.Second * 3)
}

// StartWithContainerNetwork starts a new container using an existing container network
func (c *Container) StartWithContainerNetwork(t *testing.T, other *Container, dockerArgs ...string) {
	args := []string{"run", "--rm", "--privileged", "-d", "--network=container:" + other.container}
	args = append(args, dockerArgs...)
	args = append(args, c.image)
	args = append(args, c.args...)
	result := icmd.RunCommand(dockerCli.path, args...).Assert(t, icmd.Success)
	c.container = strings.Trim(result.Stdout(), " \r\n")
	time.Sleep(time.Second * 3)
	c.parentContainer = other.container
}

// Stop terminates this container
func (c *Container) Stop(t *testing.T) {
	icmd.RunCommand(dockerCli.path, "stop", c.container).Assert(t, icmd.Success)
}

// StopNoFail terminates this container
func (c *Container) StopNoFail() {
	icmd.RunCommand(dockerCli.path, "stop", c.container)
}

// GetAddress returns the host:port this container listens on
func (c *Container) GetAddress(t *testing.T) string {
	if c.address != "" {
		return c.address
	}
	container := c.parentContainer
	if container == "" {
		container = c.container
	}
	result := icmd.RunCommand(dockerCli.path, "port", container, strconv.Itoa(c.privatePort)).Assert(t, icmd.Success)
	c.address = fmt.Sprintf("127.0.0.1:%v", strings.Trim(strings.Split(result.Stdout(), ":")[1], " \r\n"))
	return c.address
}

// GetPrivateAddress returns the host:port this container listens on
func (c *Container) GetPrivateAddress(t *testing.T) string {
	container := c.parentContainer
	if container == "" {
		container = c.container
	}
	result := icmd.RunCommand(dockerCli.path, "inspect", container, "-f", "{{.NetworkSettings.IPAddress}}").Assert(t, icmd.Success)
	return fmt.Sprintf("%s:%d", strings.TrimSpace(result.Stdout()), c.privatePort)
}

func (c *Container) Logs(t *testing.T) func() string {
	return func() string {
		return icmd.RunCommand(dockerCli.path, "logs", c.container).Assert(t, icmd.Success).Combined()
	}
}
