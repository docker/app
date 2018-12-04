package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/spf13/cobra"
	survey "gopkg.in/AlecAivazis/survey.v1"
	yaml "gopkg.in/yaml.v2"

	"github.com/deis/duffle/pkg/bundle"
	"github.com/deis/duffle/pkg/credentials"
	"github.com/deis/duffle/pkg/duffle/home"
)

const credentialGenerateHelp = `Generate credentials from a CNAB bundle

This reads a bundle.json file's credential requirements and generates a stub credentialset.
The given name becomes the name of the credentialset.

If a bundle is given, the bundle may be fetched (unless there is a cached copy),
and will then be examined. If the '-f' flag is specified, though, it will read the
bundle.json supplied.

The generated credentials will all be initialed to stub values, and should be edited
to reflect the true values.

The newly created credential set will be added to the credentialsets, though users
will still need to edit that file to set the appropriate values.
`

func newCredentialGenerateCmd(out io.Writer) *cobra.Command {
	bundleFile := ""
	var (
		insecure bool
		dryRun   bool
		noPrompt bool
	)
	cmd := &cobra.Command{
		Use:     "generate NAME [BUNDLE]",
		Aliases: []string{"gen"},
		Short:   "generate a credentialset from a bundle",
		Long:    credentialGenerateHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			bf, err := getBundleFileFromCredentialsArg(args, bundleFile, out, insecure)
			if err != nil {
				return err
			}
			csName := args[0]

			bun, err := loadBundle(bf, insecure)
			if err != nil {
				return err
			}

			generator := genCredentialSurvey
			if noPrompt {
				generator = genEmptyCredentials
			}

			creds, err := genCredentialSet(csName, bun.Credentials, generator)
			if err != nil {
				return err
			}
			data, err := yaml.Marshal(creds)
			if err != nil {
				return err
			}

			if dryRun {
				fmt.Fprintf(out, "%v", string(data))
				return nil
			}

			dest := filepath.Join(home.Home(homePath()).Credentials(), csName+".yaml")
			return ioutil.WriteFile(dest, data, 0600)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&bundleFile, "file", "f", "", "path to bundle.json")
	f.BoolVarP(&insecure, "insecure", "k", false, "do not verify the bundle (INSECURE)")
	f.BoolVar(&dryRun, "dry-run", false, "show prompts and result, but don't create credential set")
	f.BoolVarP(&noPrompt, "no-prompt", "q", false, "do not prompt for input, but generate a stub credentialset")

	return cmd
}

type credentialAnswers struct {
	Source string `survey:"source"`
	Value  string `survey:"value"`
}

const (
	questionValue   = "specific value"
	questionEnvVar  = "environment variable"
	questionPath    = "file path"
	questionCommand = "shell command"
)

type credentialGenerator func(name string) (credentials.CredentialStrategy, error)

func genCredentialSet(name string, creds map[string]bundle.Location, fn credentialGenerator) (credentials.CredentialSet, error) {
	cs := credentials.CredentialSet{
		Name: name,
	}
	cs.Credentials = []credentials.CredentialStrategy{}

	for name := range creds {
		c, err := fn(name)
		if err != nil {
			return cs, err
		}
		cs.Credentials = append(cs.Credentials, c)
	}

	return cs, nil
}

func genEmptyCredentials(name string) (credentials.CredentialStrategy, error) {
	return credentials.CredentialStrategy{
		Name:   name,
		Source: credentials.Source{Value: "EMPTY"},
	}, nil
}

func genCredentialSurvey(name string) (credentials.CredentialStrategy, error) {
	questions := []*survey.Question{
		{
			Name: "source",
			Prompt: &survey.Select{
				Message: fmt.Sprintf("Choose a source for %q", name),
				Options: []string{questionValue, questionEnvVar, questionPath, questionCommand},
				Default: "environment variable",
			},
		},
		{
			Name: "value",
			Prompt: &survey.Input{
				Message: fmt.Sprintf("Enter a value for %q", name),
			},
		},
	}
	c := credentials.CredentialStrategy{Name: name}
	answers := &credentialAnswers{}

	if err := survey.Ask(questions, answers); err != nil {
		return c, err
	}

	c.Source = credentials.Source{}
	switch answers.Source {
	case questionValue:
		c.Source.Value = answers.Value
	case questionEnvVar:
		c.Source.EnvVar = answers.Value
	case questionPath:
		c.Source.Path = answers.Value
	case questionCommand:
		c.Source.Command = answers.Value
	}
	return c, nil
}

func getBundleFileFromCredentialsArg(args []string, bundleFile string, w io.Writer, insecure bool) (string, error) {
	switch {
	case len(args) < 1:
		return "", errors.New("This command requires at least one argument: NAME (name for the credentialset). It also requires a BUNDLE (CNAB bundle name) or file (using -f)\nValid inputs:\n\t$ duffle credentials generate NAME BUNDLE\n\t$ duffle credentials generate NAME -f path-to-bundle.json")
	case len(args) == 2 && bundleFile != "":
		return "", errors.New("please use either -f or specify a BUNDLE, but not both")
	case len(args) < 2 && bundleFile == "":
		return "", errors.New("required arguments are NAME (name for the credentialset) and BUNDLE (CNAB bundle name) or file")
	case len(args) == 2:
		return getBundleFilepath(args[1], insecure)
	}
	return bundleFile, nil
}
