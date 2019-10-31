package commands

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/docker/app/internal"
	appstore "github.com/docker/app/internal/store"
	"github.com/docker/cli/cli/command"
	contextstore "github.com/docker/cli/cli/context/store"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"
	"github.com/pkg/errors"
)

type credentialSetOpt func(b *bundle.Bundle, creds credentials.Set) error

func addNamedCredentialSets(credStore appstore.CredentialStore, namedCredentialsets []string) credentialSetOpt {
	return func(_ *bundle.Bundle, creds credentials.Set) error {
		for _, file := range namedCredentialsets {
			var (
				c   *credentials.CredentialSet
				err error
			)
			// Check the credentialset locally first, then try in the credential store
			if _, e := os.Stat(file); e == nil {
				c, err = credentials.Load(file)
			} else {
				c, err = credStore.Read(file)
				if os.IsNotExist(err) {
					err = e
				}
			}
			if err != nil {
				return err
			}
			values, err := c.Resolve()
			if err != nil {
				return err
			}
			if err := creds.Merge(values); err != nil {
				return err
			}
		}
		return nil
	}
}

func parseCommandlineCredential(c string) (string, string, error) {
	split := strings.SplitN(c, "=", 2)
	if len(split) != 2 || split[0] == "" {
		return "", "", errors.Errorf("failed to parse %q as a credential name=value", c)
	}
	name := split[0]
	value := split[1]
	return name, value, nil
}

func addCredentials(strcreds []string) credentialSetOpt {
	return func(_ *bundle.Bundle, creds credentials.Set) error {
		for _, c := range strcreds {
			name, value, err := parseCommandlineCredential(c)
			if err != nil {
				return err
			}
			if err := creds.Merge(credentials.Set{
				name: value,
			}); err != nil {
				return err
			}
		}
		return nil
	}
}

func addDockerCredentials(contextName string, store contextstore.Store) credentialSetOpt {
	// docker desktop contexts require some rewriting for being used within a container
	store = internal.DockerDesktopAwareStore{Store: store}
	return func(_ *bundle.Bundle, creds credentials.Set) error {
		if contextName != "" {
			data, err := ioutil.ReadAll(contextstore.Export(contextName, store))
			if err != nil {
				return err
			}
			creds[internal.CredentialDockerContextName] = string(data)
		}
		return nil
	}
}

func addRegistryCredentials(shouldPopulate bool, dockerCli command.Cli) credentialSetOpt {
	return func(b *bundle.Bundle, creds credentials.Set) error {
		if _, ok := b.Credentials[internal.CredentialRegistryName]; !ok {
			return nil
		}

		registryCreds := map[string]types.AuthConfig{}
		if shouldPopulate {
			for _, img := range b.Images {
				named, err := reference.ParseNormalizedNamed(img.Image)
				if err != nil {
					return err
				}
				info, err := registry.ParseRepositoryInfo(named)
				if err != nil {
					return err
				}
				key := registry.GetAuthConfigKey(info.Index)
				if _, ok := registryCreds[key]; !ok {
					registryCreds[key] = command.ResolveAuthConfig(context.Background(), dockerCli, info.Index)
				}
			}
		}
		registryCredsJSON, err := json.Marshal(registryCreds)
		if err != nil {
			return err
		}
		creds[internal.CredentialRegistryName] = string(registryCredsJSON)
		return nil
	}
}

func prepareCredentialSet(b *bundle.Bundle, opts ...credentialSetOpt) (map[string]string, error) {
	creds := map[string]string{}
	for _, op := range opts {
		if err := op(b, creds); err != nil {
			return nil, err
		}
	}

	_, requiresDockerContext := b.Credentials[internal.CredentialDockerContextName]
	_, hasDockerContext := creds[internal.CredentialDockerContextName]

	// FIXME not sure what this mean if we don't have --target-context
	if requiresDockerContext && !hasDockerContext {
		return nil, errors.New("no target context specified. Use --target-context= or DOCKER_TARGET_CONTEXT= to define it")
	}

	return creds, nil
}
