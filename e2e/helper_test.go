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
	net2 "k8s.io/apimachinery/pkg/util/net"
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

	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()

	// The dind doesn't have the cnab-app-base image so we save it in order to load it later
	saveCmd := icmd.Cmd{Command: dockerCli.Command("save", fmt.Sprintf("docker/cnab-app-base:%s", internal.Version), "-o", tmpDir.Join("cnab-app-base.tar.gz"))}
	icmd.RunCmd(saveCmd).Assert(t, icmd.Success)

	// Busybox is used in a few e2e test, let's pre-load it
	cmd.Command = dockerCli.Command("pull", "busybox:1.30.1")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
	saveCmd = icmd.Cmd{Command: dockerCli.Command("save", "busybox:1.30.1", "-o", tmpDir.Join("busybox.tar.gz"))}
	icmd.RunCmd(saveCmd).Assert(t, icmd.Success)

	// we have a difficult constraint here:
	// - the registry must be reachable from the client side (for cnab-to-oci, which does not use the docker daemon to access the registry)
	// - the registry must be reachable from the dind daemon on the same address/port
	// - the installer image need to target the same docker context (dind) as the client, while running on default (or another) context, which means we can't use 'localhost'
	// Solution found is: use host external IP (not loopback) so accessing from within installer container will reach the right container

	registry := NewContainer("registry:2", 5000)
	registry.Start(t, "-e", "REGISTRY_VALIDATION_MANIFESTS_URLS_ALLOW=[^http]",
		"-e", "REGISTRY_HTTP_ADDR=0.0.0.0:5000")
	defer registry.StopNoFail()
	registryAddress := registry.GetAddress(t)

	swarm := NewContainer("docker:19.03.3-dind", 2375, "--insecure-registry", registryAddress)
	swarm.Start(t, "-e", "DOCKER_TLS_CERTDIR=") // Disable certificate generate on DinD startup
	defer swarm.Stop(t)
	swarmAddress := swarm.GetAddress(t)

	cmd.Command = dockerCli.Command("context", "create", "swarm-context", "--docker", fmt.Sprintf(`"host=tcp://%s"`, swarmAddress), "--default-stack-orchestrator", "swarm")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	cmd.Env = append(cmd.Env, "DOCKER_CONTEXT=swarm-context", "DOCKER_INSTALLER_CONTEXT=swarm-context")
	// Initialize the swarm
	cmd.Command = dockerCli.Command("swarm", "init")
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
	// Load the needed base cnab image into the swarm docker engine
	cmd.Command = dockerCli.Command("load", "-i", tmpDir.Join("cnab-app-base.tar.gz"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
	// Pre-load busybox image used by a few e2e tests
	cmd.Command = dockerCli.Command("load", "-i", tmpDir.Join("busybox.tar.gz"))
	icmd.RunCmd(cmd).Assert(t, icmd.Success)

	info := dindSwarmAndRegistryInfo{
		configuredCmd:   cmd,
		registryAddress: registryAddress,
		swarmAddress:    swarmAddress,
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
	ip := c.getIP(t)
	port := c.getPort(t)
	c.address = fmt.Sprintf("%s:%v", ip, port)
	return c.address
}

func (c *Container) getPort(t *testing.T) string {
	result := icmd.RunCommand(dockerCli.path, "port", c.container, strconv.Itoa(c.privatePort)).Assert(t, icmd.Success)
	port := strings.Trim(strings.Split(result.Stdout(), ":")[1], " \r\n")
	return port
}

var host string

func (c *Container) getIP(t *testing.T) string {
	if host != "" {
		return host
	}
	ip, err := net2.ChooseHostInterface()
	assert.NilError(t, err)
	host = ip.String()
	return host
}

func (c *Container) Logs(t *testing.T) func() string {
	return func() string {
		return icmd.RunCommand(dockerCli.path, "logs", c.container).Assert(t, icmd.Success).Combined()
	}
}
