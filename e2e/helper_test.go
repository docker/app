package e2e

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/app/internal"
	"github.com/jackpal/gateway"
	"gotest.tools/assert"
	"gotest.tools/fs"
	"gotest.tools/icmd"
	"k8s.io/apimachinery/pkg/util/wait"
)

// readFile returns the content of the file at the designated path normalizing
// line endings by removing any \r.
func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := ioutil.ReadFile(path)
	assert.NilError(t, err, "missing '"+path+"' file")
	return strings.Replace(string(content), "\r", "", -1)
}

func getHostIPAddress() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", errors.New("No IP address detected")
}

type OrchestratorAndRegistryInfo struct {
	orchestratorName    string
	orchestratorAddress string
	registryAddress     string
	configuredCmd       icmd.Cmd
	configDir           string
	tmpDir              *fs.Dir
	cleanup             func()
	stopRegistry        func()
	registryLogs        func() string
	dockerCmd           func(...string) string
	execCmd             func(...string) string
	localCmd            func(...string) string
}

func (info *OrchestratorAndRegistryInfo) LoadImageFromWeb(tag string, url string) error {
	filename := filepath.Base(url)
	err := info.Download(info.tmpDir.Join(filename), url)
	if err != nil {
		return err
	}
	combined := info.dockerCmd("load", "-q", "-i", info.tmpDir.Join(filename))
	digest := ""
	for _, line := range strings.Split(combined, "\n") {
		if strings.Contains(line, "sha256:") {
			digest = strings.Split(line, "sha256:")[1]
		}
	}
	if digest == "" {
		return errors.New("Image digest not found in docker load's stdout")
	}
	// tag image
	digest = strings.Trim(digest, " \r\n")
	info.dockerCmd("tag", digest, tag)
	return nil
}

func (info *OrchestratorAndRegistryInfo) Download(filename string, url string) error {
	client := http.Client{Timeout: time.Minute * 5}
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return errors.New("Page not found: " + url)
	}
	// Create the file
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()
	// Write the body to file
	_, err = io.Copy(out, res.Body)
	return err
}

func extractZIP(archive string, binary string, filename string) error {
	zipReader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	for _, file := range zipReader.Reader.File {
		if filename == file.Name {
			zippedFile, err := file.Open()
			if err != nil {
				return err
			}
			defer zippedFile.Close()
			binaryFile, err := os.OpenFile(
				binary,
				os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
				file.Mode(),
			)
			if err != nil {
				return err
			}
			defer binaryFile.Close()
			_, err = io.Copy(binaryFile, zippedFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (info *OrchestratorAndRegistryInfo) ExtractBinaryFromArchive(binary string, archive string, archiveFilename string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()

	if strings.HasSuffix(archive, ".zip") {
		return extractZIP(archive, binary, archiveFilename)
	}

	var fileReader io.ReadCloser = f
	// just in case we are reading a tar.gz file, add a filter to handle gzipped file
	if strings.HasSuffix(archive, ".gz") {
		if fileReader, err = gzip.NewReader(f); err != nil {
			return err
		}
		defer fileReader.Close()
	}
	tarReader := tar.NewReader(fileReader)
	for {
		file, err := tarReader.Next()
		if err != nil {
			if err != io.EOF {
				return err
			}
			return nil
		}
		if file.Typeflag != tar.TypeReg {
			continue
		}
		if archiveFilename == file.Name {
			// handle normal file
			writer, err := os.Create(binary)
			if err != nil {
				return err
			}
			_, err = io.Copy(writer, tarReader)
			if err != nil {
				return err
			}
			err = os.Chmod(binary, os.FileMode(file.Mode))
			if err != nil {
				return err
			}
			writer.Close()
		}
	}
}

func initInfoAndRegistry(t *testing.T) *OrchestratorAndRegistryInfo {
	cmd, cleanup := dockerCli.createTestCmd()
	tmpDir := fs.NewDir(t, t.Name())

	var configDir string
	for _, val := range cmd.Env {
		if ok := strings.HasPrefix(val, "DOCKER_CONFIG="); ok {
			configDir = strings.Replace(val, "DOCKER_CONFIG=", "", 1)
		}
	}
	// Initialize the info struct
	runner := &OrchestratorAndRegistryInfo{configuredCmd: cmd, configDir: configDir, tmpDir: tmpDir}

	// Func to execute command locally
	runLocalCmd := func(params ...string) string {
		if len(params) == 0 {
			return ""
		}
		cmd := icmd.Command(params[0], params[1:]...)
		cmd.Env = runner.configuredCmd.Env
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

	runner.localCmd = runLocalCmd
	runner.dockerCmd = runDockerCmd
	runner.execCmd = func(params ...string) string {
		args := append([]string{"docker", "exec", "-t", runner.orchestratorName}, params...)
		return runLocalCmd(args...)
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
	hostIP, err := getHostIPAddress()
	assert.NilError(t, err)
	registry := NewContainer("registry:2", 5000)
	registry.Start(t, "-p", fmt.Sprintf(`%s:5000:5000`, hostIP), "-e", "REGISTRY_VALIDATION_MANIFESTS_URLS_ALLOW=[^http]",
		"-e", "REGISTRY_HTTP_ADDR=0.0.0.0:5000")
	//defer registry.StopNoFail()
	registryAddress := registry.GetAddress(t)
	// Initialize the info struct
	runner.registryAddress = registryAddress
	runner.stopRegistry = registry.StopNoFail
	runner.registryLogs = registry.Logs(t)

	runner.cleanup = func() {
		runner.stopRegistry()
		runner.tmpDir.Remove()
		cleanup()
	}
	return runner
}

func runWithDindSwarmAndRegistry(t *testing.T, todo func(OrchestratorAndRegistryInfo)) {
	runner := initInfoAndRegistry(t)
	defer runner.cleanup()

	tmpDir := runner.tmpDir
	swarm := NewContainer("docker:19.03.3-dind", 2375, "--insecure-registry", runner.registryAddress)
	swarm.Start(t, "--name", "dind", "-e", "DOCKER_TLS_CERTDIR=", "-P") // Disable certificate generate on DinD startup
	defer swarm.Stop(t)
	swarmAddress := swarm.GetAddress(t)

	runner.orchestratorName = swarm.container
	runner.orchestratorAddress = swarmAddress

	runner.dockerCmd("context", "create", "swarm-context", "--docker", fmt.Sprintf(`"host=tcp://%s"`, swarmAddress), "--default-stack-orchestrator", "swarm")
	runner.configuredCmd.Env = append(runner.configuredCmd.Env, "DOCKER_CONTEXT=swarm-context", "DOCKER_INSTALLER_CONTEXT=swarm-context")

	// Initialize the swarm
	runner.dockerCmd("swarm", "init")
	// Load the needed base cnab image into the swarm docker engine
	runner.dockerCmd("load", "-i", tmpDir.Join("cnab-app-base.tar.gz"))
	// Pre-load busybox image used by a few e2e tests
	runner.dockerCmd("load", "-i", tmpDir.Join("busybox.tar.gz"))

	todo(*runner)
}

func runWithKindAndRegistry(t *testing.T, todo func(OrchestratorAndRegistryInfo)) {
	runner := initInfoAndRegistry(t)

	extension := ""
	archive := ".tar.gz"
	if runtime.GOOS == "windows" {
		extension = ".exe"
		archive = ".zip"
	}
	// paths to binaries used during install
	kind := runner.tmpDir.Join("kind" + extension)
	kubectl := runner.tmpDir.Join("kubectl" + extension)
	helm := runner.tmpDir.Join("helm" + extension)
	conk := runner.tmpDir.Join(fmt.Sprintf("installer-%s%s", runtime.GOOS, extension))

	// detect host's platform
	err := runner.Download(kind, fmt.Sprintf(`https://github.com/kubernetes-sigs/kind/releases/download/v0.6.0/kind-%s-amd64`, runtime.GOOS))
	assert.NilError(t, err)
	// make kind binary executable
	err = os.Chmod(kind, os.FileMode(0111))
	assert.NilError(t, err)
	//get hosts's ip address
	ipaddress, err := getHostIPAddress()
	assert.NilError(t, err)
	kindconf := []byte(fmt.Sprintf(`kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
networking:
  apiServerAddress: %s
containerdConfigPatches: 
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."%s"]
    endpoint = ["http://%s"]
`, ipaddress, runner.registryAddress, runner.registryAddress))

	// write kind config file
	assert.NilError(t, ioutil.WriteFile(runner.tmpDir.Join("kind.conf"), kindconf, os.FileMode(0644)))

	runner.localCmd(kind, "create", "cluster", "--name", "kindcluster", "--config", runner.tmpDir.Join("kind.conf"), "--kubeconfig", runner.tmpDir.Join("kube.conf"))
	// add kube config to env for kubectl
	runner.configuredCmd.Env = append(runner.configuredCmd.Env, "KUBECONFIG="+runner.tmpDir.Join("kube.conf"))
	runner.orchestratorName = "kindcluster-control-plane"
	// download and install kubectl, helm and compose-on-kubernetes
	err = runner.Download(kubectl, fmt.Sprintf(`https://storage.googleapis.com/kubernetes-release/release/v1.16.0/bin/%s/amd64/kubectl%s`, runtime.GOOS, extension))
	assert.NilError(t, err)
	err = os.Chmod(kubectl, os.FileMode(0111))
	assert.NilError(t, err)
	time.Sleep(time.Second * 60)

	output := runner.localCmd(kubectl, `create`, `namespace`, `compose`)
	checkContains(t, output, []string{`namespace/compose created`})

	// wait until all containers are in running state
	waitReady := func() {
		areRunning := func(s []string) bool {
			if len(s) == 0 {
				return true
			}
			if s[0] != "Running" {
				return false
			}
			for i := 1; i < len(s); i++ {
				if s[i] != s[0] {
					return false
				}
			}
			return true
		}
		err := wait.Poll(time.Second*2, time.Second*240, func() (bool, error) {
			output := runner.localCmd(kubectl, `get`, `pods`, `-o=jsonpath={.items[*].status.phase}`, `--all-namespaces`)
			containerStatus := strings.Split(strings.Trim(output, `\n`), ` `)
			return areRunning(containerStatus), nil
		})
		assert.NilError(t, err)
	}
	waitReady()

	output = runner.localCmd(kubectl, `-n`, `kube-system`, `create`, `serviceaccount`, `tiller`)
	checkContains(t, output, []string{`serviceaccount/tiller created`})
	output = runner.localCmd(kubectl, `-n`, `kube-system`, `create`, `clusterrolebinding`, `tiller`, `--clusterrole`, `cluster-admin`, `--serviceaccount`, `kube-system:tiller`)
	checkContains(t, output, []string{`clusterrolebinding.rbac.authorization.k8s.io/tiller created`})
	waitReady()
	// download and install helm
	err = runner.Download(runner.tmpDir.Join("helm"+archive), fmt.Sprintf(`https://get.helm.sh/helm-v2.16.1-%s-amd64%s`, runtime.GOOS, archive))
	assert.NilError(t, err)
	err = runner.ExtractBinaryFromArchive(helm, runner.tmpDir.Join("helm"+archive), fmt.Sprintf("%s-amd64/helm%s", runtime.GOOS, extension))
	assert.NilError(t, err)
	err = os.Chmod(helm, os.FileMode(0111))
	assert.NilError(t, err)
	runner.localCmd(helm, `init`, `--service-account`, `tiller`)
	time.Sleep(time.Second * 30)
	waitReady()

	output = runner.localCmd(helm, `install`, `--name`, `etcd-operator`, `stable/etcd-operator`, `--namespace`, `compose`)
	checkContains(t, output, []string{`1. etcd-operator deployed.`})
	time.Sleep(time.Second * 5)
	waitReady()

	// write compose file for etcd deployment
	data, err := ioutil.ReadFile(filepath.Join("testdata", "compatibility", "compose-etcd.yaml"))
	assert.NilError(t, err)
	assert.NilError(t, ioutil.WriteFile(runner.tmpDir.Join("compose-etcd.yaml"), data, os.FileMode(0644)))
	// deploy etcd for compose on kube
	runner.localCmd(kubectl, `apply`, `-f`, runner.tmpDir.Join("compose-etcd.yaml"))

	time.Sleep(time.Second * 10)
	waitReady()

	// download compose-on-kube binary
	err = runner.Download(conk, fmt.Sprintf(`https://github.com/docker/compose-on-kubernetes/releases/download/v0.5.0-alpha1/installer-%s%s`, runtime.GOOS, extension))
	assert.NilError(t, err)
	err = os.Chmod(conk, os.FileMode(0111))
	assert.NilError(t, err)
	// Install Compose on Kube
	output = runner.localCmd(conk, `-namespace=compose`, `-etcd-servers=http://compose-etcd-client:2379`)
	checkContains(t, output, []string{`Controller: image: `})
	time.Sleep(time.Second * 5)
	waitReady()

	// setup docker context
	runner.dockerCmd(`context`, `create`, `kind-context`, `--docker`, `"host=unix:///var/run/docker.sock"`, `--default-stack-orchestrator`,
		`kubernetes`, `--kubernetes`, fmt.Sprintf(`"config-file=%s"`, runner.tmpDir.Join("kube.conf")))
	runner.configuredCmd.Env = append(runner.configuredCmd.Env, "DOCKER_CONTEXT=kind-context", "DOCKER_INSTALLER_CONTEXT=kind-context")

	cleanAll := func() {
		runner.dockerCmd(`rm`, `--force`, `--volumes`, runner.orchestratorName)
		runner.cleanup()
	}
	defer cleanAll()
	todo(*runner)

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
	args := []string{"run", "--rm", "--privileged", "-d"}
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
