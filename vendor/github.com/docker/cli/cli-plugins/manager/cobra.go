package manager

import (
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

const (
	// CommandAnnotationPlugin is added to every stub command added by
	// AddPluginCommandStubs with the value "true" and so can be
	// used to distinguish plugin stubs from regular commands.
	CommandAnnotationPlugin = "com.docker.cli.plugin"

	// CommandAnnotationPluginVendor is added to every stub command
	// added by AddPluginCommandStubs and contains the vendor of
	// that plugin.
	CommandAnnotationPluginVendor = "com.docker.cli.plugin.vendor"

	// CommandAnnotationPluginInvalid is added to any stub command
	// added by AddPluginCommandStubs for an invalid command (that
	// is, one which failed it's candidate test) and contains the
	// reason for the failure.
	CommandAnnotationPluginInvalid = "com.docker.cli.plugin-invalid"
)

// AddPluginCommandStubs adds a stub cobra.Commands for each plugin
// (optionally including invalid ones). The command stubs will have
// several annotations added, see `CommandAnnotationPlugin*`.
func AddPluginCommandStubs(dockerCli command.Cli, cmd *cobra.Command, includeInvalid bool) error {
	plugins, err := ListPlugins(dockerCli, cmd)
	if err != nil {
		return err
	}
	for _, p := range plugins {
		if !includeInvalid && p.Err != nil {
			continue
		}
		vendor := p.Vendor
		if vendor == "" {
			vendor = "unknown"
		}
		annotations := map[string]string{
			CommandAnnotationPlugin:       "true",
			CommandAnnotationPluginVendor: vendor,
		}
		if p.Err != nil {
			annotations[CommandAnnotationPluginInvalid] = p.Err.Error()
		}
		cmd.AddCommand(&cobra.Command{
			Use:         p.Name,
			Short:       p.ShortDescription,
			Run:         func(_ *cobra.Command, _ []string) {},
			Annotations: annotations,
		})
	}
	return nil
}
