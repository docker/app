package context

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/kubernetes"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newSetKubernetesEndpointCommand(dockerCli command.Cli) *cobra.Command {
	opts := &kubernetesEndpointOptions{}
	cmd := &cobra.Command{
		Use:   "set-kubernetes-endpoint <context> [options]",
		Short: "Reset the kubernetes endpoint of a context",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			endpoint, err := opts.toEndpoint(name)
			if err != nil {
				return errors.Wrap(err, "unable to create kubernetes endpoint config")
			}
			if endpoint == nil {
				// remove
				ctxRaw, err := dockerCli.ContextStore().GetContextMetadata(name)
				if err != nil {
					return err
				}
				delete(ctxRaw.Endpoints, kubernetes.KubernetesEndpointKey)
				if err := dockerCli.ContextStore().CreateOrUpdateContext(name, ctxRaw); err != nil {
					return err
				}
				return dockerCli.ContextStore().ResetContextEndpointTLSMaterial(name, kubernetes.KubernetesEndpointKey, nil)
			}
			return kubernetes.Save(dockerCli.ContextStore(), *endpoint)
		},
	}

	opts.addFlags(cmd.Flags(), "")
	return cmd
}
