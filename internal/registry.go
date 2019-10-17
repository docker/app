package internal

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/command"
	"github.com/sirupsen/logrus"
)

// InsecureRegistriesFromEngine reads the registry configuration from the daemon and returns
// a list of all insecure ones.
func InsecureRegistriesFromEngine(dockerCli command.Cli) ([]string, error) {
	registries := []string{}

	info, err := dockerCli.Client().Info(context.Background())
	if err != nil {
		return nil, fmt.Errorf("could not get docker info: %v", err)
	}

	for _, reg := range info.RegistryConfig.IndexConfigs {
		if !reg.Secure {
			registries = append(registries, reg.Name)
		}
	}

	logrus.Debugf("insecure registries: %v", registries)

	return registries, nil
}
