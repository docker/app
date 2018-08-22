package main

import (
	"context"
	"io"

	"github.com/docker/app/internal"
	"github.com/docker/app/internal/com"
	"github.com/docker/app/internal/packager"
	"github.com/docker/cli/cli"
	"github.com/docker/docker/pkg/archive"
	"github.com/spf13/cobra"
)

var (
	initComposeFile string
	initDescription string
	initMaintainers []string
	initSingleFile  bool
)

// initCmd represents the init command
func initCmd(fs com.FrontServiceClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init <app-name> [-c <compose-file>] [-d <description>] [-m name:email ...]",
		Short: "Start building a Docker application",
		Long:  `Start building a Docker application. Will automatically detect a docker-compose.yml file in the current directory.`,
		Args:  cli.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := packager.Init(args[0], initComposeFile, initDescription, initMaintainers, initSingleFile); err != nil {
				return err
			}
			dirname := internal.DirNameFromAppName(args[0])
			reader, err := archive.Tar(dirname, archive.Uncompressed)
			if err != nil {
				return err
			}
			defer reader.Close()
			untarDirClient, err := fs.UntarDir(context.Background())
			if err != nil {
				return err
			}
			defer untarDirClient.CloseAndRecv()
			buffer := make([]byte, 4096)
			for {
				read, err := reader.Read(buffer)
				if err == io.EOF {
					return nil
				}
				if err != nil {
					return err
				}
				if err = untarDirClient.Send(&com.UntarPacket{
					Data: buffer[:read],
					Dest: dirname,
				}); err != nil {
					return err
				}
			}
		},
	}
	cmd.Flags().StringVarP(&initComposeFile, "compose-file", "c", "", "Initial Compose file (optional)")
	cmd.Flags().StringVarP(&initDescription, "description", "d", "", "Initial description (optional)")
	cmd.Flags().StringArrayVarP(&initMaintainers, "maintainer", "m", []string{}, "Maintainer (name:email) (optional)")
	cmd.Flags().BoolVarP(&initSingleFile, "single-file", "s", false, "Create a single-file application")
	return cmd
}
