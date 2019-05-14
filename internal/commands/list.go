package commands

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"

	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
)

type listOptions struct {
	targetContext string
	allContexts   bool
}

func listCmd(dockerCli command.Cli) *cobra.Command {
	var opts listOptions

	cmd := &cobra.Command{
		Use:     "list [OPTIONS]",
		Short:   "List the installations and their last known installation result",
		Aliases: []string{"ls"},
		Args:    cli.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.allContexts && opts.targetContext != "" {
				return errors.New("--all-contexts and --target-context flags cannot be used at the same time")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(dockerCli, opts)
		},
	}
	cmd.Flags().StringVar(&opts.targetContext, "target-context", "", "List installations on this context")
	cmd.Flags().BoolVar(&opts.allContexts, "all-contexts", false, "List installations on all contexts")

	return cmd
}

func runList(dockerCli command.Cli, opts listOptions) error {
	var contexts []string
	if opts.allContexts {
		// List all the contexts from the context store
		contextsMeta, err := dockerCli.ContextStore().List()
		if err != nil {
			return fmt.Errorf("failed to list contexts: %s", err)
		}
		for _, cm := range contextsMeta {
			contexts = append(contexts, cm.Name)
		}
		// Add a CONTEXT column
		listColumns = append(listColumns, installationColumn{"CONTEXT", func(context string, _ *store.Installation) string { return context }})
	} else {
		// Resolve the current or the specified target context
		contexts = append(contexts, getTargetContext(opts.targetContext, dockerCli.CurrentContext()))
	}
	return printInstallations(dockerCli.Out(), config.Dir(), contexts)
}

type installationColumn struct {
	header string
	value  func(c string, i *store.Installation) string
}

var (
	listColumns = []installationColumn{
		{"INSTALLATION", func(_ string, i *store.Installation) string { return i.Name }},
		{"APPLICATION", func(_ string, i *store.Installation) string {
			return fmt.Sprintf("%s (%s)", i.Bundle.Name, i.Bundle.Version)
		}},
		{"LAST ACTION", func(_ string, i *store.Installation) string { return i.Result.Action }},
		{"RESULT", func(_ string, i *store.Installation) string { return i.Result.Status }},
		{"CREATED", func(_ string, i *store.Installation) string { return units.HumanDuration(time.Since(i.Created)) }},
		{"MODIFIED", func(_ string, i *store.Installation) string { return units.HumanDuration(time.Since(i.Modified)) }},
		{"REFERENCE", func(_ string, i *store.Installation) string { return prettyPrintReference(i.Reference) }},
	}
)

func printInstallations(out io.Writer, configDir string, contexts []string) error {
	w := tabwriter.NewWriter(out, 0, 0, 1, ' ', 0)
	printHeaders(w)

	for _, context := range contexts {
		installations, err := getInstallations(context, configDir)
		if err != nil {
			return err
		}
		for _, installation := range installations {
			printValues(w, context, installation)
		}
	}

	return w.Flush()
}

func printHeaders(w io.Writer) {
	var headers []string
	for _, column := range listColumns {
		headers = append(headers, column.header)
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))
}

func printValues(w io.Writer, context string, installation *store.Installation) {
	var values []string
	for _, column := range listColumns {
		values = append(values, column.value(context, installation))
	}
	fmt.Fprintln(w, strings.Join(values, "\t"))
}

func getInstallations(targetContext, configDir string) ([]*store.Installation, error) {
	appstore, err := store.NewApplicationStore(configDir)
	if err != nil {
		return nil, err
	}
	installationStore, err := appstore.InstallationStore(targetContext)
	if err != nil {
		return nil, err
	}
	installationNames, err := installationStore.List()
	if err != nil {
		return nil, err
	}
	installations := make([]*store.Installation, len(installationNames))
	for i, name := range installationNames {
		installation, err := installationStore.Read(name)
		if err != nil {
			return nil, err
		}
		installations[i] = installation
	}
	// Sort installations with last modified first
	sort.Slice(installations, func(i, j int) bool {
		return installations[i].Modified.After(installations[j].Modified)
	})
	return installations, nil
}

func prettyPrintReference(ref string) string {
	if ref == "" {
		return ""
	}
	r, err := reference.Parse(ref)
	if err != nil {
		return ref
	}
	return reference.FamiliarString(r)
}
