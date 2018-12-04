package context

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type contextScopeOptions struct {
	description              string
	defaultStackOrchestrator string
}

func newSetOptionsCommand(dockerCli command.Cli) *cobra.Command {
	o := &contextScopeOptions{}
	cmd := &cobra.Command{
		Use:   "set-options <name> [options]",
		Short: "set common options of a context",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rawMeta, err := dockerCli.ContextStore().GetContextMetadata(args[0])
			if err != nil {
				return err
			}
			meta, err := command.GetContextMetadata(rawMeta)
			if err != nil {
				return err
			}
			if o.defaultStackOrchestrator != "" {
				if meta.StackOrchestrator, err = command.NormalizeOrchestrator(o.defaultStackOrchestrator); err != nil {
					return errors.Wrap(err, "unable to parse default-stack-orchestrator")
				}
			}
			if o.description != "" {
				meta.Description = o.description
			}
			command.SetContextMetadata(&rawMeta, meta)
			return dockerCli.ContextStore().CreateOrUpdateContext(args[0], rawMeta)
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&o.description, "description", "", "set the description of the context")
	flags.StringVar(&o.defaultStackOrchestrator, "default-stack-orchestrator", "", "set the default orchestrator for stack operations if different to the default one, to use with this context (swarm|kubernetes|all)")
	return cmd
}
