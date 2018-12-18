package main

import (
	"encoding/json"
	"testing"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/docker/api/types"
	"gotest.tools/assert"
)

type registryConfigMock struct {
	command.Cli
	configFile *configfile.ConfigFile
}

func (r *registryConfigMock) ConfigFile() *configfile.ConfigFile {
	return r.configFile
}

func TestShareRegistryCreds(t *testing.T) {
	cases := []struct {
		name       string
		shareCreds bool
		stored     map[string]types.AuthConfig
		expected   map[string]types.AuthConfig
		images     map[string]bundle.Image
	}{
		{
			name:       "no-share",
			shareCreds: false,
			stored: map[string]types.AuthConfig{
				"my-registry.com": types.AuthConfig{
					Username: "test",
					Password: "test",
				},
			},
			expected: map[string]types.AuthConfig{},
			images: map[string]bundle.Image{
				"component1": bundle.Image{
					BaseImage: bundle.BaseImage{
						Image: "my-registry.com/ns/repo:tag",
					},
				},
			},
		},
		{
			name:       "share",
			shareCreds: true,
			stored: map[string]types.AuthConfig{
				"my-registry.com": types.AuthConfig{
					Username: "test",
					Password: "test",
				},
				"my-registry2.com": types.AuthConfig{
					Username: "test",
					Password: "test",
				},
			},
			expected: map[string]types.AuthConfig{
				"my-registry.com": types.AuthConfig{
					Username: "test",
					Password: "test",
				}},
			images: map[string]bundle.Image{
				"component1": bundle.Image{
					BaseImage: bundle.BaseImage{
						Image: "my-registry.com/ns/repo:tag",
					},
				},
			},
		},
		{
			name:       "share-missing",
			shareCreds: true,
			stored: map[string]types.AuthConfig{
				"my-registry2.com": types.AuthConfig{
					Username: "test",
					Password: "test",
				},
			},
			expected: map[string]types.AuthConfig{
				"my-registry.com": types.AuthConfig{}},
			images: map[string]bundle.Image{
				"component1": bundle.Image{
					BaseImage: bundle.BaseImage{
						Image: "my-registry.com/ns/repo:tag",
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			creds, err := prepareCredentialSet("", nil, &bundle.Bundle{
				Credentials: map[string]bundle.Location{"docker.registry-creds": {}},
				Images:      c.images,
			}, nil, c.shareCreds, &registryConfigMock{configFile: &configfile.ConfigFile{
				AuthConfigs: c.stored,
			}})
			assert.NilError(t, err)
			var result map[string]types.AuthConfig
			assert.NilError(t, json.Unmarshal([]byte(creds["docker.registry-creds"]), &result))
			assert.DeepEqual(t, c.expected, result)
		})
	}
}
