package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/containerd/containerd/platforms"
	"github.com/deislabs/cnab-go/bundle"
	"github.com/docker/app/internal/log"
	"github.com/docker/app/types/metadata"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cnab-to-oci/remotes"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/docker/docker/registry"
	"github.com/morikuni/aec"
	ocischemav1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

type pushOptions struct {
	tag          string
	platforms    []string
	allPlatforms bool
}

func pushCmd(dockerCli command.Cli) *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:     "push [APP_NAME] --tag TARGET_REFERENCE [OPTIONS]",
		Short:   "Push an application package to a registry",
		Example: `$ docker app push myapp --tag myrepo/myapp:mytag`,
		Args:    cli.RequiresMaxArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return checkFlags(cmd.Flags(), opts)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(dockerCli, firstOrEmpty(args), opts)
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&opts.tag, "tag", "t", "", "Target registry reference (default: <name>:<version> from metadata)")
	flags.StringSliceVar(&opts.platforms, "platform", []string{"linux/amd64"}, "For multi-arch service images, push the specified platforms")
	flags.BoolVar(&opts.allPlatforms, "all-platforms", false, "If present, push all platforms")
	return cmd
}

func runPush(dockerCli command.Cli, name string, opts pushOptions) error {
	defer muteDockerCli(dockerCli)()
	// Get the bundle
	bndl, ref, err := resolveReferenceAndBundle(dockerCli, name)
	if err != nil {
		return err
	}
	// Retag invocation image if needed
	retag, err := shouldRetagInvocationImage(metadata.FromBundle(bndl), bndl, opts.tag, ref)
	if err != nil {
		return err
	}
	if retag.shouldRetag {
		logrus.Debugf(`Retagging invocation image "%q"`, retag.invocationImageRef.String())
		if err := retagInvocationImage(dockerCli, bndl, retag.invocationImageRef.String()); err != nil {
			return err
		}
	}
	// Push the invocation image
	if err := pushInvocationImage(dockerCli, retag); err != nil {
		return err
	}
	// Push the bundle
	return pushBundle(dockerCli, opts, bndl, retag)
}

func resolveReferenceAndBundle(dockerCli command.Cli, name string) (*bundle.Bundle, string, error) {
	bundleStore, err := prepareBundleStore()
	if err != nil {
		return nil, "", err
	}

	bndl, ref, err := resolveBundle(dockerCli, bundleStore, name, false)
	if err != nil {
		return nil, "", err
	}
	if err := bndl.Validate(); err != nil {
		return nil, "", err
	}
	return bndl, ref, err
}

func pushInvocationImage(dockerCli command.Cli, retag retagResult) error {
	logrus.Debugf("Pushing the invocation image %q", retag.invocationImageRef)
	repoInfo, err := registry.ParseRepositoryInfo(retag.invocationImageRef)
	if err != nil {
		return err
	}
	encodedAuth, err := command.EncodeAuthToBase64(command.ResolveAuthConfig(context.Background(), dockerCli, repoInfo.Index))
	if err != nil {
		return err
	}
	reader, err := dockerCli.Client().ImagePush(context.Background(), retag.invocationImageRef.String(), types.ImagePushOptions{
		RegistryAuth: encodedAuth,
	})
	if err != nil {
		return errors.Wrapf(err, "starting push of %q", retag.invocationImageRef.String())
	}
	defer reader.Close()
	if err := jsonmessage.DisplayJSONMessagesStream(reader, ioutil.Discard, 0, false, nil); err != nil {
		return errors.Wrapf(err, "pushing to %q", retag.invocationImageRef.String())
	}
	return nil
}

func pushBundle(dockerCli command.Cli, opts pushOptions, bndl *bundle.Bundle, retag retagResult) error {
	insecureRegistries, err := insecureRegistriesFromEngine(dockerCli)
	if err != nil {
		return errors.Wrap(err, "could not retrive insecure registries")
	}
	resolver := remotes.CreateResolver(dockerCli.ConfigFile(), insecureRegistries...)
	var display fixupDisplay = &plainDisplay{out: os.Stdout}
	if term.IsTerminal(os.Stdout.Fd()) {
		display = &interactiveDisplay{out: os.Stdout}
	}
	fixupOptions := []remotes.FixupOption{
		remotes.WithEventCallback(display.onEvent),
	}
	if platforms := platformFilter(opts); len(platforms) > 0 {
		fixupOptions = append(fixupOptions, remotes.WithComponentImagePlatforms(platforms))
	}
	// bundle fixup
	if err := remotes.FixupBundle(context.Background(), bndl, retag.cnabRef, resolver, fixupOptions...); err != nil {
		return errors.Wrapf(err, "fixing up %q for push", retag.cnabRef)
	}
	// push bundle manifest
	logrus.Debugf("Pushing the bundle %q", retag.cnabRef)
	descriptor, err := remotes.Push(log.WithLogContext(context.Background()), bndl, retag.cnabRef, resolver, true, withAppAnnotations)
	if err != nil {
		return errors.Wrapf(err, "pushing to %q", retag.cnabRef)
	}
	fmt.Fprintf(os.Stdout, "Successfully pushed bundle to %s. Digest is %s.\n", retag.cnabRef.String(), descriptor.Digest)
	return nil
}

func withAppAnnotations(index *ocischemav1.Index) error {
	if index.Annotations == nil {
		index.Annotations = make(map[string]string)
	}
	index.Annotations[DockerAppFormatAnnotation] = DockerAppFormatCNAB
	index.Annotations[DockerTypeAnnotation] = DockerTypeApp
	return nil
}

func platformFilter(opts pushOptions) []string {
	if opts.allPlatforms {
		return nil
	}
	return opts.platforms
}

func retagInvocationImage(dockerCli command.Cli, bndl *bundle.Bundle, newName string) error {
	err := dockerCli.Client().ImageTag(context.Background(), bndl.InvocationImages[0].Image, newName)
	if err != nil {
		return err
	}
	bndl.InvocationImages[0].Image = newName
	return nil
}

type retagResult struct {
	shouldRetag        bool
	cnabRef            reference.Named
	invocationImageRef reference.Named
}

func shouldRetagInvocationImage(meta metadata.AppMetadata, bndl *bundle.Bundle, tagOverride, bundleRef string) (retagResult, error) {
	// Use the bundle reference as a tag override
	if tagOverride == "" && bundleRef != "" {
		tagOverride = bundleRef
	}
	imgName := tagOverride
	var err error
	if imgName == "" {
		imgName, err = makeCNABImageName(meta.Name, meta.Version, "")
		if err != nil {
			return retagResult{}, err
		}
	}
	cnabRef, err := reference.ParseNormalizedNamed(imgName)
	if err != nil {
		return retagResult{}, errors.Wrap(err, imgName)
	}
	if _, digested := cnabRef.(reference.Digested); digested {
		return retagResult{}, errors.Errorf("%s: can't push to a digested reference", cnabRef)
	}
	cnabRef = reference.TagNameOnly(cnabRef)
	expectedInvocationImageRef, err := reference.ParseNormalizedNamed(reference.TagNameOnly(cnabRef).String() + "-invoc")
	if err != nil {
		return retagResult{}, errors.Wrap(err, reference.TagNameOnly(cnabRef).String()+"-invoc")
	}
	currentInvocationImageRef, err := reference.ParseNormalizedNamed(bndl.InvocationImages[0].Image)
	if err != nil {
		return retagResult{}, errors.Wrap(err, bndl.InvocationImages[0].Image)
	}
	return retagResult{
		cnabRef:            cnabRef,
		invocationImageRef: expectedInvocationImageRef,
		shouldRetag:        expectedInvocationImageRef.String() != currentInvocationImageRef.String(),
	}, nil
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

func checkFlags(flags *pflag.FlagSet, opts pushOptions) error {
	if opts.allPlatforms && flags.Changed("all-platforms") && flags.Changed("platform") {
		return fmt.Errorf("--all-plaforms and --plaform flags cannot be used at the same time")
	}
	return nil
}
