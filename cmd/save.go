package cmd

import (
	"fmt"
	"os"

	"github.com/docker/lunchbox/packager"
	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save <app-name>",
	Short: "Save the application to docker (in preparation for push)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if saveTag == "" {
			saveTag = "latest"
		}
		err := packager.Save(args[0], savePrefix, saveTag)
		if err != nil {
			fmt.Printf("%v\n", err)
			os.Exit(1)
		}
	},
}

var (
	savePrefix string
	saveTag    string
)

func init() {
	rootCmd.AddCommand(saveCmd)
	saveCmd.Flags().StringVarP(&savePrefix, "prefix", "p", "", "prefix to use")
	saveCmd.Flags().StringVarP(&saveTag, "tag", "t", "latest", "tag to use")
}
