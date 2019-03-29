package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/docker/app/internal"
)

type cnabAction func(string) error

var (
	cnabActions = map[string]cnabAction{
		"install":                  installAction,
		"upgrade":                  installAction, // upgrade is implemented as reinstall.
		"uninstall":                uninstallAction,
		internal.ActionStatusName:  statusAction,
		internal.ActionInspectName: inspectAction,
		internal.ActionRenderName:  renderAction,
	}
)

func getCnabAction() (cnabAction, string, error) {
	// CNAB_ACTION should always be set. but in future we want to have
	// claim-less actions. So we don't fail if no installation is set
	actionName, ok := os.LookupEnv("CNAB_ACTION")
	if !ok {
		return nil, "", errors.New("no CNAB action specified")
	}
	action, ok := cnabActions[actionName]
	if !ok {
		return nil, "", fmt.Errorf("action %q not supported", actionName)
	}
	return action, actionName, nil
}

func main() {
	action, actionName, err := getCnabAction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while parsing cnab operation: %s", err)
		os.Exit(1)
	}
	instanceName := os.Getenv("CNAB_INSTALLATION_NAME")
	if err := action(instanceName); err != nil {
		fmt.Fprintf(os.Stderr, "Action %q failed: %s", actionName, err)
		os.Exit(1)
	}
}
