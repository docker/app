package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gosuri/uitable"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/deis/duffle/pkg/credentials"
	"github.com/deis/duffle/pkg/duffle/home"
)

type credentialListCmd struct {
	out  io.Writer
	home home.Home
	long bool
}

func newCredentialListCmd(w io.Writer) *cobra.Command {

	list := &credentialListCmd{out: w}

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "list credential sets",
		RunE: func(cmd *cobra.Command, args []string) error {
			list.home = home.Home(homePath())
			return list.run()
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&list.long, "long", "l", false, "output longer listing format")

	return cmd
}

func (ls *credentialListCmd) run() error {
	credentialPath := ls.home.Credentials()
	creds := findCredentialSets(credentialPath)

	if ls.long {
		table := uitable.New()
		table.MaxColWidth = 80
		table.Wrap = true

		table.AddRow("NAME", "PATH")
		for _, cred := range creds {
			table.AddRow(cred.name, cred.path)
		}

		fmt.Fprintln(ls.out, table)
		return nil
	}

	for _, item := range creds {
		fmt.Fprintln(ls.out, item.name)
	}
	return nil
}

type credListItem struct {
	name string
	path string
}

func findCredentialSets(dir string) []credListItem {
	creds := []credListItem{}

	log.Debugf("Traversing credentials directory (%s) for credential sets", dir)

	filepath.Walk(dir, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !f.IsDir() {
			log.Debugf("Loading credential set from %s", path)
			credSet, err := credentials.Load(path)
			if err != nil {
				log.Debugf("Unable to load credential set from %s:\n%s", path, err)
				return nil
			}

			log.Debugf("Successfully loaded credential set %s from %s", credSet.Name, path)
			creds = append(creds, credListItem{name: credSet.Name, path: path})
		}
		return nil
	})

	return creds
}
