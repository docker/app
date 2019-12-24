package commands

import (
	"encoding/json"
	"testing"

	"github.com/cnabio/cnab-go/bundle"
	"github.com/docker/app/internal"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config/configfile"
	"github.com/docker/cli/cli/config/types"
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
				"my-registry.com": {
					Username: "test",
					Password: "test",
				},
			},
			expected: map[string]types.AuthConfig{},
			images: map[string]bundle.Image{
				"component1": {
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
				"my-registry.com": {
					Username: "test",
					Password: "test",
				},
				"my-registry2.com": {
					Username: "test",
					Password: "test",
				},
			},
			expected: map[string]types.AuthConfig{
				"my-registry.com": {
					Username: "test",
					Password: "test",
				}},
			images: map[string]bundle.Image{
				"component1": {
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
				"my-registry2.com": {
					Username: "test",
					Password: "test",
				},
			},
			expected: map[string]types.AuthConfig{
				"my-registry.com": {}},
			images: map[string]bundle.Image{
				"component1": {
					BaseImage: bundle.BaseImage{
						Image: "my-registry.com/ns/repo:tag",
					},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			creds, err := prepareCredentialSet(
				&bundle.Bundle{
					Credentials: map[string]bundle.Credential{internal.CredentialRegistryName: {}},
					Images:      c.images,
				},
				addNamedCredentialSets(nil, nil),
				addDockerCredentials("", nil),
				addRegistryCredentials(c.shareCreds, &registryConfigMock{configFile: &configfile.ConfigFile{
					AuthConfigs: c.stored,
				}}))
			assert.NilError(t, err)
			var result map[string]types.AuthConfig
			assert.NilError(t, json.Unmarshal([]byte(creds[internal.CredentialRegistryName]), &result))
			assert.DeepEqual(t, c.expected, result)
		})
	}
}

func TestParseCommandlineCredential(t *testing.T) {
	for _, tc := range []struct {
		in   string
		n, v string
		err  string // either err or n+v are non-""
	}{
		{in: "", err: `failed to parse "" as a credential name=value`},
		{in: "A", err: `failed to parse "A" as a credential name=value`},
		{in: "=B", err: `failed to parse "=B" as a credential name=value`},
		{in: "A=", n: "A", v: ""},
		{in: "A=B", n: "A", v: "B"},
		{in: "A==", n: "A", v: "="},
		{in: "A=B=C", n: "A", v: "B=C"},
	} {
		n := tc.in
		if n == "" {
			n = "«empty»"
		}
		t.Run(n, func(t *testing.T) {
			n, v, err := parseCommandlineCredential(tc.in)
			if tc.err != "" {
				assert.Error(t, err, tc.err)
			} else {
				assert.NilError(t, err)
				assert.Equal(t, tc.n, n)
				assert.Equal(t, tc.v, v)
			}
		})
	}
}
