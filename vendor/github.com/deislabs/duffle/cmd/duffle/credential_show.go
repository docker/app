package main

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/duffle/home"

	"github.com/ghodss/yaml"

	"github.com/spf13/cobra"
)

const credentialShowDesc = `
This command will fetch the credential set with the given name and prints the contents of the file.
`

type credentialShowCmd struct {
	name       string
	home       home.Home
	out        io.Writer
	unredacted bool
}

func newCredentialShowCmd(w io.Writer) *cobra.Command {
	show := &credentialShowCmd{out: w}

	cmd := &cobra.Command{
		Use:   "show [NAME]",
		Short: "show credential set",
		Long:  credentialShowDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			show.home = home.Home(homePath())
			show.name = args[0]
			return show.run()
		},
	}
	cmd.Flags().BoolVar(&show.unredacted, "unredacted", false, "Print the secret values without redacting them")
	return cmd
}

func (sh *credentialShowCmd) run() error {
	cs, err := findCredentialSet(sh.home.Credentials(), sh.name)
	if err != nil {
		return err
	}
	return sh.printCredentials(*cs)
}

func (sh *credentialShowCmd) printCredentials(cs credentials.CredentialSet) error {
	if !sh.unredacted {
		// Do not modify the passed credentials
		creds := make([]credentials.CredentialStrategy, len(cs.Credentials))
		for i, cred := range cs.Credentials {
			if cred.Source.Value != "" {
				cred.Source.Value = "REDACTED"
			}
			creds[i] = cred
		}
		cs.Credentials = creds
	}

	b, err := yaml.Marshal(cs.Name)
	if err != nil {
		return err
	}
	fmt.Fprintf(sh.out, "name: %s", string(b))
	b, err = yaml.Marshal(cs.Credentials)
	if err != nil {
		return err
	}
	fmt.Fprintf(sh.out, "credentials:\n%s", string(b))

	return nil
}

func findCredentialSet(dir, name string) (*credentials.CredentialSet, error) {
	return credentials.Load(filepath.Join(dir, fmt.Sprintf("%s.yaml", name)))
}
