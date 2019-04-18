package commands

import (
	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli"
	"github.com/spf13/cobra"
)

var splitOutputDir string

func splitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "split [APP_NAME] [--output OUTPUT_DIRECTORY]",
		Short:   "Split a single-file Docker Application definition into the directory format",
		Example: `$ docker app split myapp.dockerapp --output myapp-directory.dockerapp`,
		Args:    cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			extractedApp, err := packager.Extract(firstOrEmpty(args))
			if err != nil {
				return err
			}
			defer extractedApp.Cleanup()
			inPlace := false
			if splitOutputDir == "" {
				splitOutputDir, inPlace = handleInPlace(extractedApp)
			}
			if err := packager.Split(extractedApp, splitOutputDir); err != nil {
				return err
			}
			if inPlace {
				return removeAndRename(splitOutputDir, extractedApp.Path)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&splitOutputDir, "output", "o", "", "Output directory (default: in-place)")
	return cmd
}
