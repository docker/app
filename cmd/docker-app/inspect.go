package main

import (
	"errors"
	"io/ioutil"

	"github.com/deislabs/duffle/pkg/action"
	"github.com/deislabs/duffle/pkg/claim"
	"github.com/docker/app/types/parameters"
	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	cliopts "github.com/docker/cli/opts"
	"github.com/spf13/cobra"
)

var (
	inspectParametersFile []string
	inspectEnv            []string
	inspectInsecure       bool
)

// inspectCmd represents the inspect command
func inspectCmd(dockerCli command.Cli) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect [<app-name>] [-s key=value...] [-f parameters-file...]",
		Short: "Shows metadata, parameters and a summary of the compose file for a given application",
		Args:  cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			muteDockerCli(dockerCli)
			appname := firstOrEmpty(args)
			bndl, err := resolveBundle(dockerCli, "", appname, inspectInsecure)
			if err != nil {
				return err
			}
			if bndl.Actions == nil {
				return errors.New(`specified bundle has no "inspect" action`)
			}
			if _, ok := bndl.Actions["inspect"]; !ok {
				return errors.New(`specified bundle has no "inspect" action`)
			}
			s, err := parameters.LoadFiles(inspectParametersFile)
			if err != nil {
				return err
			}
			d := cliopts.ConvertKVStringsToMap(inspectEnv)
			overrides, err := parameters.FromFlatten(d)
			if err != nil {
				return err
			}
			if s, err = parameters.Merge(s, overrides); err != nil {
				return err
			}
			settingValues := s.Flatten()
			c, err := claim.New("inspect")
			if err != nil {
				return err
			}
			driverImpl, err := prepareDriver(dockerCli)
			if err != nil {
				return err
			}
			c.Bundle = bndl
			c.Parameters = stringsKVToStringInterface(settingValues)
			a := &action.RunCustom{
				Action: "inspect",
				Driver: driverImpl,
			}
			return a.Run(c, map[string]string{"docker.context": ""}, dockerCli.Out())
		},
	}
	cmd.Flags().StringArrayVarP(&inspectParametersFile, "parameters-files", "f", []string{}, "Override with parameters from files")
	cmd.Flags().StringArrayVarP(&inspectEnv, "set", "s", []string{}, "Override parameters values")
	cmd.Flags().BoolVar(&inspectInsecure, "insecure", false, "Use insecure registry, without SSL")
	return cmd
}

func muteDockerCli(dockerCli command.Cli) {
	dockerCli.SetOut(command.NewOutStream(ioutil.Discard))
	dockerCli.SetErr(ioutil.Discard)
}
