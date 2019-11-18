package commands

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	units "github.com/docker/go-units"
	"github.com/spf13/cobra"
)

var (
	listColumns = []struct {
		header string
		value  func(i *store.Installation) string
	}{
		{"RUNNING APP", func(i *store.Installation) string { return i.Name }},
		{"APP NAME", func(i *store.Installation) string { return fmt.Sprintf("%s (%s)", i.Bundle.Name, i.Bundle.Version) }},
		{"LAST ACTION", func(i *store.Installation) string { return i.Result.Action }},
		{"RESULT", func(i *store.Installation) string { return i.Result.Status }},
		{"CREATED", func(i *store.Installation) string {
			return fmt.Sprintf("%s ago", units.HumanDuration(time.Since(i.Created)))
		}},
		{"MODIFIED", func(i *store.Installation) string {
			return fmt.Sprintf("%s ago", units.HumanDuration(time.Since(i.Modified)))
		}},
		{"REFERENCE", func(i *store.Installation) string { return i.Reference }},
	}
)

func listCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Short:   "List running Apps",
		Aliases: []string{"list"},
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(dockerCli)
		},
	}

	return cmd
}

func runList(dockerCli command.Cli) error {
	installations, err := getInstallations(dockerCli.CurrentContext(), config.Dir())
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(dockerCli.Out(), 0, 0, 1, ' ', 0)
	printHeaders(w)

	for _, installation := range installations {
		printValues(w, installation)
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

func printValues(w io.Writer, installation *store.Installation) {
	var values []string
	for _, column := range listColumns {
		values = append(values, column.value(installation))
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
