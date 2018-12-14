package image

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/docker/cli/cli/command"
	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/registry"
	reg "github.com/genuinetools/reg/registry"
	"github.com/opencontainers/go-digest"
)

// MediaTypeCnabConfig is the config mediatype for a Cnab Image manifest
const MediaTypeCnabConfig = schema2.MediaTypeImageConfig //"application/vnd.oci.cnab.image.v1beta1+pgp"

// PushBundle pushes a signed bundle as a special image manifest in a registry
func PushBundle(ctx context.Context, cli command.Cli, nonSSL bool, signedBundle []byte, ref string) (string, error) {
	named, repoName, tag, err := parseBundleReference(ref)
	if err != nil {
		return "", err
	}

	regClient, err := makeRegClient(ctx, cli, nonSSL, named)
	if err != nil {
		return "", err
	}

	configDigest, err := uploadConfigBlob(regClient, repoName, signedBundle)
	if err != nil {
		return "", err
	}
	return uploadManifest(regClient, repoName, tag, configDigest, int64(len(signedBundle)))
}

func parseBundleReference(ref string) (named reference.Named, repoName string, tag string, err error) {
	named, err = reference.ParseNormalizedNamed(ref)
	if err != nil {
		return
	}
	repoName = reference.Path(named)
	tag = reference.TagNameOnly(named).(reference.Tagged).Tag()
	return
}

func makeRegClient(ctx context.Context, cli command.Cli, nonSSL bool, named reference.Named) (*reg.Registry, error) {
	repoInfo, err := registry.ParseRepositoryInfo(named)
	if err != nil {
		return nil, err
	}
	authConfig := command.ResolveAuthConfig(ctx, cli, repoInfo.Index)
	domain := repoInfo.Index.Name
	if repoInfo.Index.Official {
		domain = registry.DefaultV2Registry.Host
	}
	return reg.New(authConfig, reg.Opt{
		Domain:   domain,
		SkipPing: true,
		NonSSL:   nonSSL,
	})
}

func uploadConfigBlob(regClient *reg.Registry, repoName string, signedBundle []byte) (string, error) {
	blobURL, token, err := initiateUpload(regClient, repoName)
	if err != nil {
		return "", err
	}
	u, err := url.Parse(blobURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Add("digest", digest.SHA256.FromBytes(signedBundle).String())
	u.RawQuery = q.Encode()
	uploadRequest, err := http.NewRequest(http.MethodPut, u.String(), bytes.NewBuffer(signedBundle))
	if err != nil {
		return "", err
	}
	uploadRequest.Header.Set("Content-Type", "application/octet-stream")
	if token != "" {
		uploadRequest.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	resp, err := regClient.Client.Do(uploadRequest)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("failed to create config blob %q, status code is %d", repoName, resp.StatusCode)
	}
	return resp.Header.Get("Docker-Content-Digest"), nil
}

func uploadManifest(regClient *reg.Registry, repoName, tag, configDigest string, bundleSize int64) (string, error) {
	man, err := schema2.FromStruct(schema2.Manifest{
		Versioned: schema2.SchemaVersion,
		Config: distribution.Descriptor{
			Digest:    digest.Digest(configDigest),
			MediaType: MediaTypeCnabConfig,
			Size:      bundleSize,
		},
	})
	if err != nil {
		return "", err
	}
	manBytes, err := man.MarshalJSON()
	if err != nil {
		return "", err
	}
	manURL := fmt.Sprintf("%s/v2/%s/manifests/%s", regClient.URL, repoName, tag)
	manRequest, err := http.NewRequest(http.MethodPut, manURL, &autoresetBuffer{buffer: manBytes})
	if err != nil {
		return "", err
	}
	manRequest.Header.Set("Content-Type", schema2.MediaTypeManifest)
	resp, err := regClient.Client.Do(manRequest)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg, err := ioutil.ReadAll(resp.Body)
		fmt.Println(string(errMsg), err)
		return "", fmt.Errorf("failed to upload manifest %q, status code is %d", repoName, resp.StatusCode)
	}
	return resp.Header.Get("Docker-Content-Digest"), nil
}

func initiateUpload(r *reg.Registry, repoName string) (string, string, error) {
	u := fmt.Sprintf("%s/v2/%s/blobs/uploads/", r.URL, repoName)
	resp, err := r.Client.Post(u, "application/octet-stream", nil)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg, err := ioutil.ReadAll(resp.Body)
		fmt.Println(string(errMsg), err)
		return "", "", fmt.Errorf("failed to initiate config upload %q, status code is %d", repoName, resp.StatusCode)
	}
	token := resp.Header.Get("Request-Token")
	location := resp.Header.Get("Location")
	return location, token, nil
}

// http transport makes two requests, one for authentication, one is the actual request.
// autoresetBuffer resets the read buffer for the second request.
type autoresetBuffer struct {
	io.Reader
	buffer []byte
}

func (a *autoresetBuffer) Read(p []byte) (n int, err error) {
	if a.Reader == nil {
		a.Reader = ioutil.NopCloser(bytes.NewBuffer(a.buffer))
	}
	n, err = a.Reader.Read(p)
	if err == io.EOF {
		a.Reader = nil
	}
	return
}
