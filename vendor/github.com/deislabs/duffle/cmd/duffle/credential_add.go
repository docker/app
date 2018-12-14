package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/deislabs/duffle/pkg/credentials"
	"github.com/deislabs/duffle/pkg/duffle/home"
)

const credentialAddDesc = `
This command takes a path to a file, validates that the file contains a valid credential set, and adds the credential sets to duffle.

It is also possible to pass in multiple paths to this command to conveniently add multiple credential sets.
`

type credentialAddCmd struct {
	paths []string
	home  home.Home
	out   io.Writer
}

func newCredentialAddCmd(w io.Writer) *cobra.Command {
	add := &credentialAddCmd{out: w}

	cmd := &cobra.Command{
		Use:   "add [PATH]",
		Short: "add one or more credential sets",
		Long:  credentialAddDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			add.home = home.Home(homePath())
			if len(args) < 1 {
				return errors.New("This command requires at least 1 argument: path to credential set")
			}
			add.paths = args

			return add.run()
		},
	}
	return cmd
}

func (add *credentialAddCmd) run() error {
	addErrors := []string{}

	for _, path := range add.paths {
		fi, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				doesNotExist := fmt.Sprintf("File (%s) does not exist", path)
				addErrors = append(addErrors, doesNotExist)
				continue
			} else {
				addErrors = append(addErrors, err.Error())
				continue
			}
		}

		if fi.IsDir() {
			dirErr := fmt.Sprintf("%s is a directory. Enter path to a credential set file", path)
			addErrors = append(addErrors, dirErr)
			continue
		} else {
			dest := filepath.Join(add.home.Credentials(), fi.Name())
			err = addCredentialSet(dest, path)
			if err != nil {
				addErrors = append(addErrors, err.Error())
				continue
			}
		}
	}

	if len(addErrors) > 0 {
		return fmt.Errorf(strings.Join(addErrors, "\n"))
	}
	return nil
}

func addCredentialSet(dest, path string) error {
	// validate file to be credential set
	cs, err := credentials.Load(path)
	if err != nil {
		return fmt.Errorf("%s is not a valid credential set", path)
	}

	if err := fileNameMatchesSetName(path, cs.Name); err != nil {
		return err
	}

	// check if it already exists
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		return fmt.Errorf("Credential set (%s) already exists", cs.Name)
	}

	if err := copyCredentialSetFile(dest, path); err != nil {
		return err
	}
	log.Debugf("Successfully added credential set: %s", cs.Name)
	return nil
}

func copyCredentialSetFile(dest, path string) error {
	from, err := os.Open(path)
	if err != nil {
		return err
	}

	defer from.Close()

	to, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	return err
}

func fileNameMatchesSetName(path, name string) error {
	base := filepath.Base(path)
	parts := strings.Split(base, ".")
	computedName := ""
	len := len(parts)
	if len > 1 {
		computedName = strings.Join(parts[0:len-1], ".")

	} else {
		return fmt.Errorf("%s does not have valid .yaml/.yml file extension", path)
	}

	if computedName != name {
		return fmt.Errorf("file name (%s) does not match credential set name (%s)", computedName, name)
	}

	return nil
}
