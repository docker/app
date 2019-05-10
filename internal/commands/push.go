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
	"github.com/docker/app/internal/packager"
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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type pushOptions struct {
	registry  registryOptions
	tag       string
	platforms []string
}

func pushCmd(dockerCli command.Cli) *cobra.Command {
	var opts pushOptions
	cmd := &cobra.Command{
		Use:     "push [APP_NAME] --tag TARGET_REFERENCE [OPTIONS]",
		Short:   "Push an application package to a registry",
		Example: `$ docker app push myapp --tag myrepo/myapp:mytag`,
		Args:    cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(dockerCli, firstOrEmpty(args), opts)
		},
	}
	flags := cmd.Flags()
	flags.StringVarP(&opts.tag, "tag", "t", "", "Target registry reference (default: <name>:<version> from metadata)")
	flags.StringSliceVar(&opts.platforms, "platform", nil, "For multi-arch service images, only push the specified platforms")
	opts.registry.addFlags(flags)
	return cmd
}

func runPush(dockerCli command.Cli, name string, opts pushOptions) error {
	defer muteDockerCli(dockerCli)()
	app, err := packager.Extract(name)
	if err != nil {
		return err
	}
	defer app.Cleanup()
	bndl, err := makeBundleFromApp(dockerCli, app, nil)
	if err != nil {
		return err
	}
	retag, err := shouldRetagInvocationImage(app.Metadata(), bndl, opts.tag)
	if err != nil {
		return err
	}
	if retag.shouldRetag {
		err := retagInvocationImage(dockerCli, bndl, retag.invocationImageRef.String())
		if err != nil {
			return err
		}
	}

	// pushing invocation image
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
		return err
	}
	defer reader.Close()
	if err = jsonmessage.DisplayJSONMessagesStream(reader, ioutil.Discard, 0, false, nil); err != nil {
		return err
	}

	resolverConfig := remotes.NewResolverConfigFromDockerConfigFile(dockerCli.ConfigFile(), opts.registry.insecureRegistries...)
	var display fixupDisplay = &plainDisplay{out: os.Stdout}
	if term.IsTerminal(os.Stdout.Fd()) {
		display = &interactiveDisplay{out: os.Stdout}
	}
	fixupOptions := []remotes.FixupOption{
		remotes.WithEventCallback(display.onEvent),
	}
	if len(opts.platforms) > 0 {
		fixupOptions = append(fixupOptions, remotes.WithComponentImagePlatforms(opts.platforms))
	}
	// bundle fixup
	err = remotes.FixupBundle(context.Background(), bndl, retag.cnabRef, resolverConfig, fixupOptions...)

	if err != nil {
		return err
	}
	// push bundle manifest
	descriptor, err := remotes.Push(context.Background(), bndl, retag.cnabRef, resolverConfig.Resolver, true)
	if err != nil {
		return err
	}
	fmt.Printf("Successfully pushed bundle to %s. Digest is %s.\n", retag.cnabRef.String(), descriptor.Digest)
	return nil
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

func shouldRetagInvocationImage(meta metadata.AppMetadata, bndl *bundle.Bundle, tagOverride string) (retagResult, error) {
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
