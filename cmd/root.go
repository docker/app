package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/docker/lunchbox/internal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "docker-app",
	Short: "Docker App Packages",
	Long: `Create, render deploy and otherwise manipulate an app package.
For most sub-commands that take an app-package as only positional argument, this
argument is optional: an app package is looked for in the current working directory.
All commands accept both compressed and uncompressed app packages.`,
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if internal.Debug {
			log.SetLevel(log.DebugLevel)
		}
		return nil
	},
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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVar(&internal.Debug, "debug", false, "Enable debug mode")

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.lunchbox.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	/*	if cfgFile != "" {
			// Use config file from the flag.
			viper.SetConfigFile(cfgFile)
		} else {
			// Find home directory.
			home, err := homedir.Dir()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			// Search config in home directory with name ".lunchbox" (without extension).
			viper.AddConfigPath(home)
			viper.SetConfigName(".lunchbox")
		}

		viper.AutomaticEnv() // read in environment variables that match

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err == nil {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}*/
}
