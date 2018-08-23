package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/packager"
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

func mergeCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge [<app-name>] [-o output_file]",
		Short: "Merge a multi-file application into a single file",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			extractedApp, err := packager.ExtractWithOrigin(firstOrEmpty(args), nil)
			if err != nil {
				return err
			}
			defer extractedApp.Cleanup()
			inPlace := mergeOutputFile == ""
			if inPlace {
				extra, err := extraFiles(extractedApp.AppName)
				if err != nil {
					return errors.Wrap(err, "error scanning application directory")
				}
				if len(extra) != 0 {
					return fmt.Errorf("refusing to overwrite %s: extra files would be deleted: %s", extractedApp.OriginalAppName, strings.Join(extra, ","))
				}
				mergeOutputFile = extractedApp.OriginalAppName + ".tmp"
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
			if err := packager.Merge(extractedApp.AppName, target); err != nil {
				return err
			}
			if mergeOutputFile != "-" {
				// Need to close for the Rename to work on windows.
				target.(io.WriteCloser).Close()
			}
			if inPlace {
				if err := os.RemoveAll(extractedApp.OriginalAppName); err != nil {
					return errors.Wrap(err, "failed to erase previous application")
				}
				if err := os.Rename(mergeOutputFile, extractedApp.OriginalAppName); err != nil {
					return errors.Wrap(err, "failed to rename new application")
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&mergeOutputFile, "output", "o", "", "Output file (default: in-place)")
	return cmd
}
