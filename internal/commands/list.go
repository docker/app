package commands

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/deislabs/cnab-go/action"
	"github.com/docker/app/internal"
	"github.com/docker/app/internal/cliopts"
	"github.com/docker/app/internal/cnab"
	"github.com/docker/app/internal/store"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/templates"
	units "github.com/docker/go-units"
	"github.com/docker/go/canonical/json"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	listColumns = []struct {
		header string
		value  func(i Installation) string
	}{
		{"RUNNING APP", func(i Installation) string { return i.Name }},
		{"APP NAME", func(i Installation) string { return fmt.Sprintf("%s (%s)", i.Bundle.Name, i.Bundle.Version) }},
		{"SERVICES", printServices},
		{"LAST ACTION", func(i Installation) string { return i.Result.Action }},
		{"RESULT", func(i Installation) string { return i.Result.Status }},
		{"CREATED", func(i Installation) string {
			return fmt.Sprintf("%s ago", units.HumanDuration(time.Since(i.Created)))
		}},
		{"MODIFIED", func(i Installation) string {
			return fmt.Sprintf("%s ago", units.HumanDuration(time.Since(i.Modified)))
		}},
		{"REFERENCE", func(i Installation) string { return i.Reference }},
	}
)

type listOptions struct {
	template string
}

func listCmd(dockerCli command.Cli, installerContext *cliopts.InstallerContextOptions) *cobra.Command {
	var opts listOptions
	cmd := &cobra.Command{
		Use:     "ls [OPTIONS]",
		Short:   "List running Apps",
		Aliases: []string{"list"},
		Args:    cli.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(dockerCli, opts, installerContext)
		},
	}

	cmd.Flags().StringVarP(&opts.template, "format", "f", "", "Format the output using the given syntax or Go template")
	cmd.Flags().SetAnnotation("format", "experimentalCLI", []string{"true"}) //nolint:errcheck
	return cmd
}

func runList(dockerCli command.Cli, opts listOptions, installerContext *cliopts.InstallerContextOptions) error {
	// initialize stores
	appstore, err := store.NewApplicationStore(config.Dir())
	if err != nil {
		return err
	}
	targetContext := dockerCli.CurrentContext()
	installationStore, err := appstore.InstallationStore(targetContext)
	if err != nil {
		return err
	}

	fetcher := &serviceFetcher{
		dockerCli:        dockerCli,
		opts:             opts,
		installerContext: installerContext,
	}
	installations, err := getInstallations(installationStore, fetcher)
	if installations == nil && err != nil {
		return err
	}
	if err != nil {
		fmt.Fprintf(dockerCli.Err(), "%s\n", err)
	}

	if opts.template == "json" {
		bytes, err := json.MarshalIndent(installations, "", "  ")
		if err != nil {
			return errors.Errorf("Failed to marshall json: %s", err)
		}
		_, err = dockerCli.Out().Write(bytes)
		return err
	}
	if opts.template != "" {
		tmpl, err := templates.Parse(opts.template)
		if err != nil {
			return errors.Errorf("Template parsing error: %s", err)
		}
		return tmpl.Execute(dockerCli.Out(), installations)
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

func printValues(w io.Writer, installation Installation) {
	var values []string
	for _, column := range listColumns {
		values = append(values, column.value(installation))
	}
	fmt.Fprintln(w, strings.Join(values, "\t"))
}

type Installation struct {
	*store.Installation
	Services appServices `json:",omitempty"`
}

func getInstallations(installationStore store.InstallationStore, fetcher ServiceFetcher) ([]Installation, error) {
	installationNames, err := installationStore.List()
	if err != nil {
		return nil, err
	}
	installations := make([]Installation, len(installationNames))
	var errs []string
	for i, name := range installationNames {
		installation, err := installationStore.Read(name)
		if err != nil {
			return nil, err
		}
		services, err := fetcher.getServices(installation)
		if err != nil {
			errs = append(errs, err.Error())
		}
		installations[i] = Installation{Installation: installation, Services: services}
	}
	// Sort installations with last modified first
	sort.Slice(installations, func(i, j int) bool {
		return installations[i].Modified.After(installations[j].Modified)
	})
	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, "\n"))
	}
	return installations, err
}

type ServiceStatus struct {
	DesiredTasks int
	RunningTasks int
}

type appServices map[string]ServiceStatus

type runningService struct {
	Spec struct {
		Name string
	}
	ServiceStatus ServiceStatus
}

type serviceFetcher struct {
	dockerCli        command.Cli
	opts             listOptions
	installerContext *cliopts.InstallerContextOptions
}

type ServiceFetcher interface {
	getServices(*store.Installation) (appServices, error)
}

func (s *serviceFetcher) getServices(installation *store.Installation) (appServices, error) {
	defer muteDockerCli(s.dockerCli)()

	// bundle without status action returns empty services
	if !hasAction(installation.Bundle, internal.ActionStatusJSONName) {
		return nil, nil
	}
	creds, err := prepareCredentialSet(installation.Bundle,
		addDockerCredentials(s.dockerCli.CurrentContext(), s.dockerCli.ContextStore()),
		addRegistryCredentials(false, s.dockerCli),
	)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	driverImpl, errBuf, err := cnab.SetupDriver(installation, s.dockerCli, s.installerContext, &buf)
	if err != nil {
		return nil, err
	}
	a := &action.RunCustom{
		Driver: driverImpl,
		Action: internal.ActionStatusJSONName,
	}
	// fetch output from status JSON action and parse it
	if err := a.Run(&installation.Claim, creds); err != nil {
		return nil, fmt.Errorf("failed to get app %q status : %s\n%s", installation.Name, err, errBuf)
	}
	var runningServices []runningService
	if err := json.Unmarshal(buf.Bytes(), &runningServices); err != nil {
		return nil, err
	}

	services := make(appServices, len(installation.Bundle.Images))
	for name := range installation.Bundle.Images {
		services[name] = getRunningService(runningServices, installation.Name, name)
	}

	return services, nil
}

func getRunningService(services []runningService, app, name string) ServiceStatus {
	for _, s := range services {
		// swarm services are prefixed by app name
		if s.Spec.Name == name || s.Spec.Name == fmt.Sprintf("%s_%s", app, name) {
			return s.ServiceStatus
		}
	}
	return ServiceStatus{}
}

func printServices(i Installation) string {
	if len(i.Services) == 0 {
		return "N/A"
	}
	var runningServices int
	for _, s := range i.Services {
		if s.RunningTasks > 0 {
			runningServices++
		}
	}
	return fmt.Sprintf("%d/%d", runningServices, len(i.Services))
}
