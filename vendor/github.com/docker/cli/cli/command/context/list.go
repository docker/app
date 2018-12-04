package context

import (
	"fmt"
	"sort"

	"vbom.ml/util/sortorder"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	ctxformatter "github.com/docker/cli/cli/command/context/formatter"
	"github.com/docker/cli/cli/command/formatter"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/kubernetes"
	"github.com/spf13/cobra"
)

type listOptions struct {
	format string
}

func newListCommand(dockerCli command.Cli) *cobra.Command {
	opts := &listOptions{}
	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Aliases: []string{"list"},
		Short:   "List contexts",
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(dockerCli, opts)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.format, "format", "", "Pretty-print contexts using a Go template")
	return cmd
}

func runList(dockerCli command.Cli, opts *listOptions) error {
	curContext := dockerCli.CurrentContext()
	contextMap, err := dockerCli.ContextStore().ListContexts()
	if err != nil {
		return err
	}
	var contexts []*ctxformatter.ClientContext
	for name, rawMeta := range contextMap {
		meta, err := command.GetContextMetadata(rawMeta)
		if err != nil {
			return err
		}
		dockerEndpoint, err := docker.EndpointFromContext(name, rawMeta)
		if err != nil {
			return err
		}
		kubernetesEndpoint := kubernetes.EndpointFromContext(name, rawMeta)
		kubEndpointText := ""
		if kubernetesEndpoint != nil {
			kubEndpointText = fmt.Sprintf("%s (%s)", kubernetesEndpoint.Host, kubernetesEndpoint.DefaultNamespace)
		}
		desc := ctxformatter.ClientContext{
			Name:               name,
			Current:            name == curContext,
			Description:        meta.Description,
			StackOrchestrator:  string(meta.StackOrchestrator),
			DockerEndpoint:     dockerEndpoint.Host,
			KubernetesEndpoint: kubEndpointText,
		}
		contexts = append(contexts, &desc)
	}
	sort.Slice(contexts, func(i, j int) bool {
		return sortorder.NaturalLess(contexts[i].Name, contexts[j].Name)
	})
	return format(dockerCli, opts, contexts)
}

func format(dockerCli command.Cli, opts *listOptions, contexts []*ctxformatter.ClientContext) error {
	format := opts.format
	if format == "" {
		format = ctxformatter.ClientContextTableFormat
	}
	contextCtx := formatter.Context{
		Output: dockerCli.Out(),
		Format: formatter.Format(format),
	}
	return ctxformatter.ClientContextWrite(contextCtx, contexts)
}
