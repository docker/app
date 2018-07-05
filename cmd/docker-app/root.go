package main

import (
	"fmt"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/cli/cli/command"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
// FIXME(vdemeester) use command.Cli interface
func newRootCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "docker-app",
		Short:        "Docker App Packages",
		Long:         `Build and deploy Docker applications.`,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if internal.Debug {
				log.SetLevel(log.DebugLevel)
			}
			return nil
		},
	}
	cmd.PersistentFlags().BoolVar(&internal.Debug, "debug", false, "Enable debug mode")
	addCommands(cmd, dockerCli)
	return cmd
}

// addCommands adds all the commands from cli/command to the root command
func addCommands(cmd *cobra.Command, dockerCli command.Cli) {
	cmd.AddCommand(
		deployCmd(dockerCli),
		helmCmd(),
		initCmd(),
		inspectCmd(dockerCli),
		lsCmd(),
		mergeCmd(dockerCli),
		pushCmd(),
		renderCmd(dockerCli),
		saveCmd(dockerCli),
		splitCmd(),
		versionCmd(dockerCli),
	)
	if internal.Experimental == "on" {
		cmd.AddCommand(
			imageAddCmd(),
			imageLoadCmd(),
			loadCmd(),
			packCmd(dockerCli),
			pullCmd(),
			unpackCmd(),
		)
	}
}

func firstOrEmpty(list []string) string {
	if len(list) != 0 {
		return list[0]
	}
	return ""
}

func parseSettings(s []string) (map[string]string, error) {
	d := make(map[string]string)
	for _, v := range s {
		kv := strings.SplitN(v, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("Missing '=' in setting '%s', expected KEY=VALUE", v)
		}
		if _, ok := d[kv[0]]; ok {
			return nil, fmt.Errorf("Duplicate command line setting: '%s'", kv[0])
		}
		d[kv[0]] = kv[1]
	}
	return d, nil
}
