package driver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CommandDriver relies upon a system command to provide a driver implementation
type CommandDriver struct {
	Name string
}

// Run executes the command
func (d *CommandDriver) Run(op *Operation) error {
	return d.exec(op)
}

// Handles executes the driver with `--handles` and parses the results
func (d *CommandDriver) Handles(dt string) bool {
	out, err := exec.Command(d.cliName(), "--handles").CombinedOutput()
	if err != nil {
		fmt.Printf("%s --handles: %s", d.cliName(), err)
		return false
	}
	types := strings.Split(string(out), ",")
	for _, tt := range types {
		if dt == strings.TrimSpace(tt) {
			return true
		}
	}
	return false
}

func (d *CommandDriver) cliName() string {
	return "duffle-" + strings.ToLower(d.Name)
}

func (d *CommandDriver) exec(op *Operation) error {
	// We need to do two things here: We need to make it easier for the
	// command to access data, and we need to make it easy for the command
	// to pass that data on to the image it invokes. So we do some data
	// duplication.

	// Construct an environment for the subprocess by cloning our
	// environment and adding in all the extra env vars.
	pairs := os.Environ()
	added := []string{}
	for k, v := range op.Environment {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
		added = append(added, k)
	}
	// DUFFLE_VARS is a list of variables we added to the env. This is to make
	// it easier for shell script drivers to clone the env vars.
	pairs = append(pairs, fmt.Sprintf("DUFFLE_VARS=%s", strings.Join(added, ",")))

	data, err := json.Marshal(op)
	if err != nil {
		return err
	}
	args := []string{}
	cmd := exec.Command(d.cliName(), args...)
	cmd.Dir, err = os.Getwd()
	if err != nil {
		return err
	}
	cmd.Env = pairs
	cmd.Stdin = bytes.NewBuffer(data)
	out, err := cmd.CombinedOutput()
	fmt.Fprintln(op.Out, string(out))
	return err
}
