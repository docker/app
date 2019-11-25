package cnab

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/docker/app/internal/cliopts"
	store2 "github.com/docker/app/internal/store"

	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/driver"
	dockerDriver "github.com/deislabs/cnab-go/driver/docker"
	"github.com/docker/app/internal"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

// BindMount
type BindMount struct {
	required bool
	endpoint string
}

const defaultSocketPath string = "/var/run/docker.sock"

func RequiredClaimBindMount(c claim.Claim, dockerCli command.Cli) (BindMount, error) {
	var specifiedOrchestrator string
	if rawOrchestrator, ok := c.Parameters[internal.ParameterOrchestratorName]; ok {
		specifiedOrchestrator = rawOrchestrator.(string)
	}

	return RequiredBindMount(dockerCli.CurrentContext(), specifiedOrchestrator, dockerCli.ContextStore())
}

// RequiredBindMount Returns the path required to bind mount when running
// the invocation image.
func RequiredBindMount(targetContextName string, targetOrchestrator string, s store.Store) (BindMount, error) {
	if targetOrchestrator == "kubernetes" {
		return BindMount{}, nil
	}

	if targetContextName == "" {
		targetContextName = "default"
	}

	// in case of docker desktop, we want to rewrite the context in cases where it targets the local swarm or Kubernetes
	s = &internal.DockerDesktopAwareStore{Store: s}

	ctxMeta, err := s.GetMetadata(targetContextName)
	if err != nil {
		return BindMount{}, err
	}
	dockerCtx, err := command.GetDockerContext(ctxMeta)
	if err != nil {
		return BindMount{}, err
	}
	if dockerCtx.StackOrchestrator == command.OrchestratorKubernetes {
		return BindMount{}, nil
	}
	dockerEndpoint, err := docker.EndpointFromContext(ctxMeta)
	if err != nil {
		return BindMount{}, err
	}

	host := dockerEndpoint.Host
	return BindMount{isDockerHostLocal(host), socketPath(host)}, nil
}

func socketPath(host string) string {
	if strings.HasPrefix(host, "unix://") {
		return strings.TrimPrefix(host, "unix://")
	}

	return defaultSocketPath
}

func isDockerHostLocal(host string) bool {
	return host == "" || strings.HasPrefix(host, "unix://") || strings.HasPrefix(host, "npipe://")
}

// prepareDriver prepares a driver per the user's request.
func prepareDriver(dockerCli command.Cli, bindMount BindMount, stdout io.Writer) (driver.Driver, *bytes.Buffer) {
	d := &dockerDriver.Driver{}
	errBuf := bytes.NewBuffer(nil)
	d.SetDockerCli(dockerCli)
	if stdout != nil {
		d.SetContainerOut(stdout)
	}
	d.SetContainerErr(errBuf)
	if bindMount.required {
		d.AddConfigurationOptions(func(config *container.Config, hostConfig *container.HostConfig) error {
			config.User = "0:0"
			mounts := []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: bindMount.endpoint,
					Target: bindMount.endpoint,
				},
			}
			hostConfig.Mounts = mounts
			return nil
		})
	}

	// Load any driver-specific config out of the environment.
	driverCfg := map[string]string{}
	for env := range d.Config() {
		if value, ok := os.LookupEnv(env); ok {
			driverCfg[env] = value
		}
	}
	d.SetConfig(driverCfg)

	return d, errBuf
}

func SetupDriver(installation *store2.Installation, dockerCli command.Cli, opts *cliopts.InstallerContextOptions, stdout io.Writer) (driver.Driver, *bytes.Buffer, error) {
	dockerCli, err := opts.SetInstallerContext(dockerCli)
	if err != nil {
		return nil, nil, err
	}
	bind, err := RequiredClaimBindMount(installation.Claim, dockerCli)
	if err != nil {
		return nil, nil, err
	}
	driverImpl, errBuf := prepareDriver(dockerCli, bind, stdout)
	return driverImpl, errBuf, nil
}
