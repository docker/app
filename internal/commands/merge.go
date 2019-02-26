package commands

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
	"github.com/docker/app/types"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var mergeOutputFile string

// Check appname directory for extra files and return them
func extraFiles(appname string) ([]string, error) {
	files, err := ioutil.ReadDir(appname)
	if err != nil {
		return nil, err
	}
	var res []string
	for _, f := range files {
		hit := false
		for _, afn := range internal.FileNames {
			if afn == f.Name() {
				hit = true
				break
			}
		}
		if !hit {
			res = append(res, f.Name())
		}
	}
	return res, nil
}

//handleInPlace returns the operation target path and if it's in-place
func handleInPlace(app *types.App) (string, bool) {
	if app.Source == types.AppSourceImage {
		return internal.DirNameFromAppName(app.Name), false
	}
	return app.Path + ".tmp", true
}

// removeAndRename removes target and rename source into target
func removeAndRename(source, target string) error {
	if err := os.RemoveAll(target); err != nil {
		return errors.Wrap(err, "failed to erase previous application")
	}
	if err := os.Rename(source, target); err != nil {
		return errors.Wrap(err, "failed to rename new application")
	}
	return nil
}

func mergeCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge [<app-name>] [-o output_file]",
		Short: "Merge a multi-file application into a single file",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			extractedApp, err := packager.Extract(firstOrEmpty(args))
			if err != nil {
				return err
			}
			defer extractedApp.Cleanup()
			inPlace := false
			if mergeOutputFile == "" {
				mergeOutputFile, inPlace = handleInPlace(extractedApp)
			}
			if inPlace {
				extra, err := extraFiles(extractedApp.Path)
				if err != nil {
					return errors.Wrap(err, "error scanning application directory")
				}
				if len(extra) != 0 {
					return fmt.Errorf("refusing to overwrite %s: extra files would be deleted: %s", extractedApp.Path, strings.Join(extra, ","))
				}
			}
			var target io.Writer
			if mergeOutputFile == "-" {
				target = dockerCli.Out()
			} else {
				target, err = os.Create(mergeOutputFile)
				if err != nil {
					return err
				}
			}
			if err := packager.Merge(extractedApp, target); err != nil {
				return err
			}
			if mergeOutputFile != "-" {
				// Need to close for the Rename to work on windows.
				target.(io.WriteCloser).Close()
			}
			if inPlace {
				return removeAndRename(mergeOutputFile, extractedApp.Path)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&mergeOutputFile, "output", "o", "", "Output file (default: in-place)")
	return cmd
}
