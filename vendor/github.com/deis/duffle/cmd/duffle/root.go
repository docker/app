package main

import (
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var verbose bool

// newRootCmd builds the root duffle command
// - outputRedirect: Optional, specify to capture all command output (stderr and stdout)
func newRootCmd(outputRedirect io.Writer) *cobra.Command {
	const usage = `The CNAB installer`

	cmd := &cobra.Command{
		Use:          "duffle",
		Short:        usage,
		SilenceUsage: true,
		Long:         usage,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if verbose {
				log.SetLevel(log.DebugLevel)
			}
			if cmd.Name() == "init" {
				return nil
			}
			err := autoInit(cmd.OutOrStdout(), false)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "pre-flight check failed: %s", err)
			}
			return err
		},
	}
	cmd.SetOutput(outputRedirect)
	outLog := cmd.OutOrStdout()

	p := cmd.PersistentFlags()
	p.StringVar(&duffleHome, "home", defaultDuffleHome(), "location of your Duffle config. Overrides $DUFFLE_HOME")
	p.BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	cmd.AddCommand(newBuildCmd(outLog))
	cmd.AddCommand(newBundleCmd(outLog))
	cmd.AddCommand(newInitCmd(outLog))
	cmd.AddCommand(newInspectCmd(outLog))
	cmd.AddCommand(newListCmd(outLog))
	cmd.AddCommand(newPullCmd(outLog))
	cmd.AddCommand(newPushCmd(outLog))
	cmd.AddCommand(newSearchCmd(outLog))
	cmd.AddCommand(newVersionCmd(outLog))
	cmd.AddCommand(newInstallCmd())
	cmd.AddCommand(newStatusCmd(outLog))
	cmd.AddCommand(newUninstallCmd())
	cmd.AddCommand(newUpgradeCmd())
	cmd.AddCommand(newRunCmd(outLog))
	cmd.AddCommand(newCredentialsCmd(outLog))
	cmd.AddCommand(newKeyCmd(outLog))
	cmd.AddCommand(newClaimsCmd(outLog))
	cmd.AddCommand(newExportCmd(outLog))
	cmd.AddCommand(newImportCmd(outLog))
	cmd.AddCommand(newCreateCmd(outLog))

	return cmd
}
