package store

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/deislabs/duffle/pkg/bundle"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
)

func storeBaseDir() string {
	return config.Path("app-bundle-store")
}
func storePath(ref reference.Named) (string, error) {
	name := ref.Name()
	// A name is safe for use as a filesystem path (it is
	// alphanumerics + "." + "/") except for the ":" used to
	// separate domain from port which is not safe on Windows.
	// Replace it with "_" which is not valid in the name.
	//
	// There can be at most 1 ":" in a valid reference so only
	// replace one -- if there are more (and this wasn't caught
	// when parsing the ref) then there will be errors when we try
	// to use this as a path later.
	name = strings.Replace(name, ":", "_", 1)

	storeDir := filepath.Join(storeBaseDir(), filepath.FromSlash(name))

	// We rely here on _ not being valid in a name meaning there can be no clashes due to nesting of repositories.
	switch t := ref.(type) {
	case reference.Digested:
		digest := t.Digest()
		storeDir = filepath.Join(storeDir, "_digests", digest.Algorithm().String(), digest.Encoded())
	case reference.Tagged:
		storeDir = filepath.Join(storeDir, "_tags", t.Tag())
	default:
		return "", errors.Errorf("%s: not tagged or digested", ref.String())
	}

	return storeDir + ".json", nil
}

// LookupOrPullBundle will fetch the given bundle from the local
// bundle store, or if it is missing from the registry, and returns
// it. Always pulls if pullRef is true. If it pulls then the local
// bundle store is updated.
func LookupOrPullBundle(dockerCli command.Cli, ref reference.Named, pullRef bool, insecureRegistries []string) (*bundle.Bundle, error) {
	path, err := storePath(ref)
	if err != nil {
		return nil, errors.Wrap(err, ref.String())
	}

	if !pullRef {
		r, err := os.Open(path)
		if err == nil {
			defer r.Close()
			bndl, err := bundle.ParseReader(r)
			return &bndl, err

		}
		if !os.IsNotExist(err) {
			return nil, errors.Wrap(err, ref.String())
		}
	}

	bndl, err := remotes.Pull(context.Background(), reference.TagNameOnly(ref), remotes.NewResolverConfigFromDockerConfigFile(dockerCli.ConfigFile(), insecureRegistries...).Resolver)
	if err != nil {
		return nil, errors.Wrap(err, ref.String())
	}

	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return nil, err
	}
	if err := bndl.WriteFile(path, 0666); err != nil {
		return nil, err
	}
	return bndl, err
}
