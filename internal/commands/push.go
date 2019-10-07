package commands

import (
	"context"
	"fmt"

	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/log"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/registry"
	"github.com/opencontainers/go-digest"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const ( // Docker specific annotations and values
	// DockerAppFormatAnnotation is the top level annotation specifying the kind of the App Bundle
	DockerAppFormatAnnotation = "io.docker.app.format"
	// DockerAppFormatCNAB is the DockerAppFormatAnnotation value for CNAB
	DockerAppFormatCNAB = "cnab"

	// DockerTypeAnnotation is the annotation that designates the type of the application
	DockerTypeAnnotation = "io.docker.type"
	// DockerTypeApp is the value used to fill DockerTypeAnnotation when targeting a docker-app
	DockerTypeApp = "app"
)

func pushCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "push APP_IMAGE[:TAG]",
		Short:   "Push an application package to a registry",
		Long:    "Push an application, including all the service images and the invocation image to a registry",
		Example: `$ docker app push myuser/myapplication:0.1.0`,
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bundleStore, err := prepareBundleStore()
			if err != nil {
				return err
			}

			pushCmd, err := newPushCommand(dockerCli, args[0], bundleStore)
			if err != nil {
				return err
			}
			return pushCmd.Run()
		},
	}

	return cmd
}

type pushCommand struct {
	dockerCli    command.Cli
	bundleStore  store.BundleStore
	appImageName string
	ref          reference.Named
	repo         string
	bundle       *bundle.Bundle
}

func newPushCommand(dockerCli command.Cli, name string, bundleStore store.BundleStore) (*pushCommand, error) {
	ref, err := reference.ParseDockerRef(name)
	if err != nil {
		return nil, err
	}

	repo := fmt.Sprintf("%s/%s", reference.Domain(ref), reference.Path(ref))

	fmt.Fprintf(dockerCli.Out(), "The push refers to repository %s\n", repo)

	bundle, err := bundleStore.Read(ref)
	if err != nil {
		return nil, fmt.Errorf("An application does not exist locally with name: %q", name)
	}

	return &pushCommand{
		dockerCli:    dockerCli,
		bundleStore:  bundleStore,
		appImageName: name,
		ref:          ref,
		repo:         repo,
		bundle:       bundle,
	}, nil
}

// Run pushes all the service and invocation images as well as the bundle
func (c *pushCommand) Run() error {
	if err := c.pushServiceImages(); err != nil {
		return err
	}

	if err := c.pushInvocationImages(); err != nil {
		return err
	}

	if err := c.pushBundle(); err != nil {
		return err
	}

	if err := c.updateLocalBundle(); err != nil {
		return err
	}

	return nil
}

func (c *pushCommand) pushServiceImages() error {
	for service, image := range c.bundle.Images {
		fmt.Fprintf(c.dockerCli.Out(), "Pushing service image %q\n", service)

		digest, err := c.pushUnderSingleTag(image.BaseImage)
		if err != nil {
			return err
		}

		image.Image = fmt.Sprintf("%s@%s", c.repo, digest.String())
		image.MediaType = schema2.MediaTypeManifest
		c.bundle.Images[service] = image
	}

	return nil
}

func (c *pushCommand) pushInvocationImages() error {
	for i, image := range c.bundle.InvocationImages {
		fmt.Fprintln(c.dockerCli.Out(), "Pushing invocation image")

		digest, err := c.pushUnderSingleTag(image.BaseImage)
		if err != nil {
			return err
		}

		image.Image = fmt.Sprintf("%s@%s", c.repo, digest.String())
		image.MediaType = schema2.MediaTypeManifest
		c.bundle.InvocationImages[i] = image
	}

	return nil
}

func (c *pushCommand) pushBundle() error {
	fmt.Fprintf(c.dockerCli.Out(), "Pushing the bundle %q\n", c.ref)

	insecureRegistries, err := insecureRegistriesFromEngine(c.dockerCli)
	if err != nil {
		return errors.Wrap(err, "could not retrieve insecure registries")
	}

	resolver := remotes.CreateResolver(c.dockerCli.ConfigFile(), insecureRegistries...)
	descriptor, err := remotes.Push(log.WithLogContext(context.Background()), c.bundle, c.ref, resolver, true, withAppAnnotations)
	if err != nil {
		return fmt.Errorf("could not push to %q", c.ref)
	}
	fmt.Fprintf(c.dockerCli.Out(), "Successfully pushed bundle to %q. Digest is %s.\n", c.ref, descriptor.Digest)
	return nil
}

func (c *pushCommand) updateLocalBundle() error {
	return c.bundleStore.Store(c.ref, c.bundle)
}

func (c *pushCommand) pushUnderSingleTag(image bundle.BaseImage) (digest.Digest, error) {
	ref := image.Digest
	if ref == "" {
		ref = image.Image
	}

	// tag everything under the same reference
	if err := c.dockerCli.Client().ImageTag(context.Background(), ref, c.ref.String()); err != nil {
		return "", err
	}

	digest, err := c.pushImage()
	if err != nil {
		return "", err
	}
	fmt.Sprintln(c.dockerCli.Out(), digest)
	return digest, err
}

func (c *pushCommand) pushImage() (digest.Digest, error) {
	logrus.Debugf("pushing image %q", c.ref.String())

	repoInfo, err := registry.ParseRepositoryInfo(c.ref)
	if err != nil {
		return "", err
	}

	encodedAuth, err := command.EncodeAuthToBase64(command.ResolveAuthConfig(context.Background(), c.dockerCli, repoInfo.Index))
	if err != nil {
		return "", err
	}

	reader, err := c.dockerCli.Client().ImagePush(context.Background(), c.ref.String(), types.ImagePushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return "", errors.Wrapf(err, "could not push to %q", c.ref.String())
	}
	defer reader.Close()
	d := digestCollector{out: c.dockerCli.Out()}
	if err := jsonmessage.DisplayJSONMessagesToStream(reader, &d, nil); err != nil {
		return "", errors.Wrapf(err, "could not push to %q", c.ref.String())
	}

	dg, err := d.Digest()
	if err == nil && dg != "" {
		return dg, nil
	}

	return "", nil
}

type digestCollector struct {
	out  *streams.Out
	last string
}

// Write implement writer.Write
func (d *digestCollector) Write(p []byte) (n int, err error) {
	d.last = string(p)
	return d.out.Write(p)
}

// Digest return the image digest collected by parsing "docker push" stdout
func (d digestCollector) Digest() (digest.Digest, error) {
	dg := digest.DigestRegexp.FindString(d.last)
	return digest.Parse(dg)
}

// FD implement stream.FD
func (d *digestCollector) FD() uintptr {
	return d.out.FD()
}

// IsTerminal implement stream.IsTerminal
func (d *digestCollector) IsTerminal() bool {
	return d.out.IsTerminal()
}

func withAppAnnotations(index *ocischemav1.Index) error {
	if index.Annotations == nil {
		index.Annotations = make(map[string]string)
	}
	index.Annotations[DockerAppFormatAnnotation] = DockerAppFormatCNAB
	index.Annotations[DockerTypeAnnotation] = DockerTypeApp
	return nil
}
