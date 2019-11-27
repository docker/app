package e2e

import (
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/app/internal"
	"github.com/jackpal/gateway"
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

type dindSwarmAndRegistryInfo struct {
	swarmAddress    string
	registryAddress string
	configuredCmd   icmd.Cmd
	configDir       string
	tmpDir          *fs.Dir
	stopRegistry    func()
	registryLogs    func() string
	dockerCmd       func(...string) string
	execCmd         func(...string) string
	localCmd        func(...string) string
}

func runWithDindSwarmAndRegistry(t *testing.T, todo func(dindSwarmAndRegistryInfo)) {
	cmd, cleanup := dockerCli.createTestCmd()
	defer cleanup()

	tmpDir := fs.NewDir(t, t.Name())
	defer tmpDir.Remove()

	var configDir string
	for _, val := range cmd.Env {
		if ok := strings.HasPrefix(val, "DOCKER_CONFIG="); ok {
			configDir = strings.Replace(val, "DOCKER_CONFIG=", "", 1)
		}
	}

	// Initialize the info struct
	runner := dindSwarmAndRegistryInfo{configuredCmd: cmd, configDir: configDir, tmpDir: tmpDir}

	// Func to execute command locally
	runLocalCmd := func(params ...string) string {
		if len(params) == 0 {
			return ""
		}
		cmd := icmd.Command(params[0], params[1:]...)
		result := icmd.RunCmd(cmd)
		result.Assert(t, icmd.Success)
		return result.Combined()
	}
	// Func to execute docker cli commands
	runDockerCmd := func(params ...string) string {
		runner.configuredCmd.Command = dockerCli.Command(params...)
		result := icmd.RunCmd(runner.configuredCmd)
		result.Assert(t, icmd.Success)
		return result.Combined()
	}

	// The dind doesn't have the cnab-app-base image so we save it in order to load it later
	runDockerCmd("save", fmt.Sprintf("docker/cnab-app-base:%s", internal.Version), "-o", tmpDir.Join("cnab-app-base.tar.gz"))

	// Busybox is used in a few e2e test, let's pre-load it
	runDockerCmd("pull", "busybox:1.30.1")
	runDockerCmd("save", "busybox:1.30.1", "-o", tmpDir.Join("busybox.tar.gz"))

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

	// Initialize the info struct
	runner.registryAddress = registryAddress
	runner.swarmAddress = swarmAddress
	runner.stopRegistry = registry.StopNoFail
	runner.registryLogs = registry.Logs(t)

	runDockerCmd("context", "create", "swarm-context", "--docker", fmt.Sprintf(`"host=tcp://%s"`, swarmAddress), "--default-stack-orchestrator", "swarm")

	runner.configuredCmd.Env = append(runner.configuredCmd.Env, "DOCKER_CONTEXT=swarm-context", "DOCKER_INSTALLER_CONTEXT=swarm-context")

	// Initialize the swarm
	runDockerCmd("swarm", "init")
	// Load the needed base cnab image into the swarm docker engine
	runDockerCmd("load", "-i", tmpDir.Join("cnab-app-base.tar.gz"))
	// Pre-load busybox image used by a few e2e tests
	runDockerCmd("load", "-i", tmpDir.Join("busybox.tar.gz"))

	runner.localCmd = runLocalCmd
	runner.dockerCmd = runDockerCmd
	runner.execCmd = func(params ...string) string {
		args := append([]string{"docker", "exec", "-t", swarm.container}, params...)
		return runLocalCmd(args...)
	}
	todo(runner)
}

func build(t *testing.T, cmd icmd.Cmd, dockerCli dockerCliCommand, ref, path string) {
	cmd.Command = dockerCli.Command("app", "build", "-t", ref, path)
	icmd.RunCmd(cmd).Assert(t, icmd.Success)
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
	// Discover default gateway
	gw, err := gateway.DiscoverGateway()
	assert.NilError(t, err)

	// Search for the interface configured on the same network as the gateway
	addrs, err := net.InterfaceAddrs()
	assert.NilError(t, err)
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			net1 := ipnet.IP.Mask(ipnet.Mask).String()
			net2 := gw.Mask(ipnet.Mask).String()
			if net1 == net2 {
				host = ipnet.IP.String()
				break
			}
		}
	}
	return host
}

func (c *Container) Logs(t *testing.T) func() string {
	return func() string {
		return icmd.RunCommand(dockerCli.path, "logs", c.container).Assert(t, icmd.Success).Combined()
	}
}
