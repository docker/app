package main

import (
	"fmt"
	"strings"

	"github.com/docker/app/internal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	commands = []*cobra.Command{
		deployCmd(),
		helmCmd(),
		initCmd(),
		inspectCmd(),
		loadCmd(),
		packCmd(),
		pullCmd(),
		pushCmd(),
		renderCmd(),
		saveCmd(),
		unpackCmd(),
		versionCmd(),
	}
	experimentalCommands = []*cobra.Command{
		imageAddCmd(),
		imageLoadCmd(),
		mergeCmd(),
		splitCmd(),
	}
)

// rootCmd represents the base command when called without any subcommands
func newRootCmd() *cobra.Command {
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
	for _, c := range commands {
		cmd.AddCommand(c)
	}
	if internal.Experimental == "on" {
		for _, c := range experimentalCommands {
			cmd.AddCommand(c)
		}
	}
	return cmd
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
