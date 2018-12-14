package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/duffle/home"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"gopkg.in/AlecAivazis/survey.v1"
)

const credentialEditDesc = `
Open an editor for editing the named credential.

Upon saving and exiting the editor, this will write the updated credential.

This uses the values of $EDITOR or $VISUAL to figure out which editor to use. If none is found,
this will default to 'vi' on UNIX-like systems and Notepad on Windows.
`

type credentialEditCmd struct {
	name string
	home home.Home
	out  io.Writer
}

func newCredentialEditCmd(w io.Writer) *cobra.Command {
	edit := &credentialEditCmd{out: w}

	cmd := &cobra.Command{
		Use:   "edit [NAME]",
		Short: "edit an existing credential set",
		Long:  credentialEditDesc,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			edit.home = home.Home(homePath())
			edit.name = args[0]
			return edit.run()
		},
	}

	return cmd
}

func (c *credentialEditCmd) run() error {
	creds, err := findCredentialSet(c.home.Credentials(), c.name)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(creds)
	if err != nil {
		return err
	}

	strdata := `# Credential fields:
# - name: NAME OF CREDENTIAL
#   source:
#     value: "A literal value"
#     env: ENV_VAR_NAME  # environment variable containing credentials
#     path: /some/path   # path to a file containing credentials
` + string(data)

	prompt := &survey.Editor{
		Message: "Edit your credentials, then save and exit.",
		Default: strdata,
		// This shows the text in the editor.
		AppendDefault: true,
		// This hides the text from the prompt.
		HideDefault: true,
	}

	var dest string
	if err := survey.AskOne(prompt, &dest, nil); err != nil {
		return err
	}

	// Validate that this works.
	newcreds := &credentials.CredentialSet{}
	if err := yaml.Unmarshal([]byte(dest), newcreds); err != nil {
		return fmt.Errorf("new credentials are malformed: %s", err)
	}

	destpath := filepath.Join(c.home.Credentials(), c.name+".yaml")
	return ioutil.WriteFile(destpath, []byte(dest), 0600)
}
