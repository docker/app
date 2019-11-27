package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/opencontainers/go-digest"

	"github.com/docker/app/internal/relocated"
	"github.com/docker/app/internal/store"

	"github.com/containerd/containerd/platforms"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/log"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/pkg/term"
	"github.com/morikuni/aec"
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
		Use:     "push APP_IMAGE",
		Short:   "Push an App image to a registry",
		Example: `$ docker app push myrepo/myapp:mytag`,
		Args:    cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(dockerCli, args[0])
		},
	}
	return cmd
}

func runPush(dockerCli command.Cli, name string) error {
	defer muteDockerCli(dockerCli)()
	bundleStore, err := prepareBundleStore()
	if err != nil {
		return err
	}

	// Get the bundle
	ref, err := reference.ParseDockerRef(name)
	if err != nil {
		return errors.Wrapf(err, "could not push %q", name)
	}

	bndl, err := resolveReferenceAndBundle(bundleStore, ref)
	if err != nil {
		return err
	}

	cnabRef := reference.TagNameOnly(ref)

	// Push the bundle
	dg, err := pushBundle(dockerCli, bndl, cnabRef)
	if err != nil {
		return errors.Wrapf(err, "could not push %q", cnabRef)
	}
	bndl.RepoDigest = dg
	_, err = bundleStore.Store(bndl, cnabRef)
	return err
}

func resolveReferenceAndBundle(bundleStore store.BundleStore, ref reference.Reference) (*relocated.Bundle, error) {
	bndl, err := bundleStore.Read(ref)
	if err != nil {
		return nil, errors.Wrapf(err, "could not push %q: no such App image", reference.FamiliarString(ref))
	}

	if err := bndl.Validate(); err != nil {
		return nil, err
	}

	return bndl, err
}

func pushBundle(dockerCli command.Cli, bndl *relocated.Bundle, cnabRef reference.Named) (digest.Digest, error) {
	insecureRegistries, err := internal.InsecureRegistriesFromEngine(dockerCli)
	if err != nil {
		return "", errors.Wrap(err, "could not retrieve insecure registries")
	}
	resolver := remotes.CreateResolver(dockerCli.ConfigFile(), insecureRegistries...)
	var display fixupDisplay = &plainDisplay{out: os.Stdout}
	if term.IsTerminal(os.Stdout.Fd()) {
		display = &interactiveDisplay{out: os.Stdout}
	}
	fixupOptions := []remotes.FixupOption{
		remotes.WithEventCallback(display.onEvent),
		remotes.WithAutoBundleUpdate(),
		remotes.WithPushImages(dockerCli.Client(), dockerCli.Out()),
		remotes.WithRelocationMap(bndl.RelocationMap),
	}
	// bundle fixup
	relocationMap, err := remotes.FixupBundle(context.Background(), bndl.Bundle, cnabRef, resolver, fixupOptions...)
	if err != nil {
		return "", errors.Wrapf(err, "fixing up %q for push", cnabRef)
	}
	bndl.RelocationMap = relocationMap
	// push bundle manifest
	logrus.Debugf("Pushing the bundle %q", cnabRef)
	descriptor, err := remotes.Push(log.WithLogContext(context.Background()), bndl.Bundle, bndl.RelocationMap, cnabRef, resolver, true, withAppAnnotations)
	if err != nil {
		return "", errors.Wrapf(err, "pushing to %q", cnabRef)
	}
	fmt.Fprintf(os.Stdout, "Successfully pushed bundle to %s. Digest is %s.\n", cnabRef, descriptor.Digest)
	return descriptor.Digest, nil
}

func withAppAnnotations(index *ocischemav1.Index) error {
	if index.Annotations == nil {
		index.Annotations = make(map[string]string)
	}
	index.Annotations[DockerAppFormatAnnotation] = DockerAppFormatCNAB
	index.Annotations[DockerTypeAnnotation] = DockerTypeApp
	return nil
}

type fixupDisplay interface {
	onEvent(remotes.FixupEvent)
}

type interactiveDisplay struct {
	out               io.Writer
	previousLineCount int
	images            []interactiveImageState
}

func (r *interactiveDisplay) onEvent(ev remotes.FixupEvent) {
	out := bytes.NewBuffer(nil)
	for i := 0; i < r.previousLineCount; i++ {
		fmt.Fprint(out, aec.NewBuilder(aec.Up(1), aec.EraseLine(aec.EraseModes.All)).ANSI)
	}
	switch ev.EventType {
	case remotes.FixupEventTypeCopyImageStart:
		r.images = append(r.images, interactiveImageState{name: ev.SourceImage})
	case remotes.FixupEventTypeCopyImageEnd:
		r.images[r.imageIndex(ev.SourceImage)].done = true
	case remotes.FixupEventTypeProgress:
		r.images[r.imageIndex(ev.SourceImage)].onProgress(ev.Progress)
	}
	r.previousLineCount = 0
	for _, s := range r.images {
		r.previousLineCount += s.print(out)
	}
	r.out.Write(out.Bytes()) //nolint:errcheck // nothing much we can do with an error to write to output.
}

func (r *interactiveDisplay) imageIndex(name string) int {
	for ix, state := range r.images {
		if state.name == name {
			return ix
		}
	}
	return 0
}

type interactiveImageState struct {
	name     string
	progress remotes.ProgressSnapshot
	done     bool
}

func (s *interactiveImageState) onProgress(p remotes.ProgressSnapshot) {
	s.progress = p
}

func (s *interactiveImageState) print(out io.Writer) int {
	if s.done {
		fmt.Fprint(out, aec.Apply(s.name, aec.BlueF))
	} else {
		fmt.Fprint(out, s.name)
	}
	fmt.Fprint(out, "\n")
	lineCount := 1

	for _, p := range s.progress.Roots {
		lineCount += printDescriptorProgress(out, &p, 1)
	}
	return lineCount
}

func printDescriptorProgress(out io.Writer, p *remotes.DescriptorProgressSnapshot, depth int) int {
	fmt.Fprint(out, strings.Repeat(" ", depth))
	name := p.MediaType
	if p.Platform != nil {
		name = platforms.Format(*p.Platform)
	}
	if len(p.Children) == 0 {
		name = fmt.Sprintf("%s...: %s", p.Digest.String()[:15], p.Action)
	}
	doneCount := 0
	for _, c := range p.Children {
		if c.Done {
			doneCount++
		}
	}
	display := name
	if len(p.Children) > 0 {
		display = fmt.Sprintf("%s [%d/%d] (%s...)", name, doneCount, len(p.Children), p.Digest.String()[:15])
	}
	if p.Done {
		display = aec.Apply(display, aec.BlueF)
	}
	if hasError(p) {
		display = aec.Apply(display, aec.RedF)
	}
	fmt.Fprintln(out, display)
	lineCount := 1
	if p.Done {
		return lineCount
	}
	for _, c := range p.Children {
		lineCount += printDescriptorProgress(out, &c, depth+1)
	}
	return lineCount
}

func hasError(p *remotes.DescriptorProgressSnapshot) bool {
	if p.Error != nil {
		return true
	}
	for _, c := range p.Children {
		if hasError(&c) {
			return true
		}
	}
	return false
}

type plainDisplay struct {
	out io.Writer
}

func (r *plainDisplay) onEvent(ev remotes.FixupEvent) {
	switch ev.EventType {
	case remotes.FixupEventTypeCopyImageStart:
		fmt.Fprintf(r.out, "Handling image %s...", ev.SourceImage)
	case remotes.FixupEventTypeCopyImageEnd:
		if ev.Error != nil {
			fmt.Fprintf(r.out, "\nFailure: %s\n", ev.Error)
		} else {
			fmt.Fprint(r.out, " done!\n")
		}
	}
}
