package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/spf13/cobra"

	"github.com/deis/duffle/pkg/bundle"
	"github.com/deis/duffle/pkg/crypto/digest"
	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/loader"
)

func newPullCmd(w io.Writer) *cobra.Command {
	const usage = `Pulls a CNAB bundle into the cache without installing it.

Example:
	$ duffle pull duffle/example:0.1.0
`

	var insecure bool
	cmd := &cobra.Command{
		Hidden: true,
		Use:    "pull",
		Short:  "pull a CNAB bundle from a repository",
		Long:   usage,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := pullBundle(args[0], insecure)
			if err != nil {
				return err
			}
			fmt.Fprintln(w, path)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&insecure, "insecure", "k", false, "Do not verify the bundle (INSECURE)")

	return cmd
}

func pullBundle(bundleName string, insecure bool) (string, error) {
	home := home.Home(homePath())
	ref, err := getReference(bundleName)
	if err != nil {
		return "", err
	}

	url, err := getBundleRepoURL(bundleName)
	if err != nil {
		return "", err
	}
	resp, err := http.Get(url.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request to %s responded with a non-200 status code: %d", url, resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("could not read bundle from remote server: %s", err)
	}

	digest, err := digest.OfBuffer(data)
	if err != nil {
		return "", err
	}

	ldr, err := getLoader(insecure)
	if err != nil {
		return "", err
	}

	bundle, err := ldr.LoadData(data)
	if err != nil {
		return "", err
	}

	bundleFilepath := filepath.Join(home.Bundles(), digest)
	if err := bundle.WriteFile(bundleFilepath, 0644); err != nil {
		return "", fmt.Errorf("failed to write bundle: %v", err)
	}

	return bundleFilepath, recordBundleReference(home, ref, digest)
}

func getLoader(insecure bool) (loader.Loader, error) {
	var load loader.Loader
	if insecure {
		load = loader.NewDetectingLoader()
	} else {
		kr, err := loadVerifyingKeyRings(homePath())
		if err != nil {
			return nil, fmt.Errorf("cannot securely load bundle: %s", err)
		}
		load = loader.NewSecureLoader(kr)
	}
	return load, nil
}

func getReference(bundleName string) (reference.NamedTagged, error) {
	var (
		name string
		ref  reference.NamedTagged
	)

	parts := strings.SplitN(bundleName, "://", 2)
	if len(parts) == 2 {
		name = parts[1]
	} else {
		name = parts[0]
	}
	normalizedRef, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return nil, fmt.Errorf("%q is not a valid bundle name: %v", name, err)
	}
	if reference.IsNameOnly(normalizedRef) {
		ref, err = reference.WithTag(normalizedRef, "latest")
		if err != nil {
			// NOTE(bacongobbler): Using the default tag *must* be valid.
			// To create a NamedTagged type with non-validated
			// input, the WithTag function should be used instead.
			panic(err)
		}
	} else {
		if taggedRef, ok := normalizedRef.(reference.NamedTagged); ok {
			ref = taggedRef
		} else {
			return nil, fmt.Errorf("unsupported image name: %s", normalizedRef.String())
		}
	}

	return ref, nil
}

func getBundleRepoURL(bundleName string) (*url.URL, error) {
	ref, err := getReference(bundleName)
	if err != nil {
		return nil, err
	}
	return repoURLFromReference(ref)
}

func repoURLFromReference(ref reference.NamedTagged) (*url.URL, error) {
	// TODO: find a way to make the proto configurable
	url := &url.URL{
		Scheme: "https",
		Host:   reference.Domain(ref),
		Path:   fmt.Sprintf("repositories/%s/tags/%s", reference.Path(ref), ref.Tag()),
	}
	return url, nil
}

func loadBundle(bundleFile string, insecure bool) (*bundle.Bundle, error) {
	l, err := getLoader(insecure)
	if err != nil {
		return nil, err
	}
	// Issue #439: Errors that come back from the loader can be
	// pretty opaque.
	var bun *bundle.Bundle
	if bun, err = l.Load(bundleFile); err != nil {
		if err.Error() == "no signature block in data" {
			return bun, errors.New("bundle is not signed")
		}
		// Dear Go, Y U NO TERNARY, kthxbye
		secflag := "secure"
		if insecure {
			secflag = "insecure"
		}
		return bun, fmt.Errorf("cannot load %s bundle: %s", secflag, err)
	}
	return bun, nil
}
