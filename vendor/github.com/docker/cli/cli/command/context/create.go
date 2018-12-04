package context

import (
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/kubernetes"
	"github.com/docker/cli/cli/context/store"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type createOptions struct {
	name                     string
	description              string
	defaultStackOrchestrator string
	docker                   dockerEndpointOptions
	kubernetes               kubernetesEndpointOptions
}

func newCreateCommand(dockerCli command.Cli) *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create <name> [options]",
		Short: "create a context",
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.name = args[0]
			return opts.process(dockerCli, dockerCli.ContextStore())
		},
	}

	opts.addFlags(cmd.Flags())
	return cmd
}

func (o *createOptions) addFlags(flags *pflag.FlagSet) {
	flags.StringVar(&o.description, "description", "", "set the description of the context")
	flags.StringVar(
		&o.defaultStackOrchestrator,
		"default-stack-orchestrator", "",
		"set the default orchestrator for stack operations if different to the default one, to use with this context (swarm|kubernetes|all)")
	o.docker.addFlags(flags, "docker-")
	o.kubernetes.addFlags(flags, "kubernetes-")
}

func (o *createOptions) process(cli command.Cli, s store.Store) error {
	if _, err := s.GetContextMetadata(o.name); !store.IsErrContextDoesNotExist(err) {
		if err != nil {
			return errors.Wrap(err, "error while getting existing contexts")
		}
		return errors.Errorf("context %q already exists", o.name)
	}
	stackOrchestrator, err := command.NormalizeOrchestrator(o.defaultStackOrchestrator)
	if err != nil {
		return errors.Wrap(err, "unable to parse default-stack-orchestrator")
	}
	dockerEP, err := o.docker.toEndpoint(cli, o.name)
	if err != nil {
		return errors.Wrap(err, "unable to create docker endpoint config")
	}
	if err := docker.Save(s, dockerEP); err != nil {
		return errors.Wrap(err, "unable to save docker endpoint config")
	}
	kubernetesEP, err := o.kubernetes.toEndpoint(o.name)
	if err != nil {
		return errors.Wrap(err, "unable to create kubernetes endpoint config")
	}
	if kubernetesEP != nil {
		if err := kubernetes.Save(s, *kubernetesEP); err != nil {
			return errors.Wrap(err, "unable to save kubernetes endpoint config")
		}
	}

	// at this point, the context should exist with endpoints configuration
	ctx, err := s.GetContextMetadata(o.name)
	if err != nil {
		return errors.Wrap(err, "error while getting context")
	}
	command.SetContextMetadata(&ctx, command.ContextMetadata{
		Description:       o.description,
		StackOrchestrator: stackOrchestrator,
	})

	return s.CreateOrUpdateContext(o.name, ctx)
}
