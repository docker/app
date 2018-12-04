package main

import (
	"errors"
	"fmt"
	"os"
)

type cnabAction string

const (
	cnabActionInstall   = cnabAction("install")
	cnabActionUninstall = cnabAction("uninstall")
	cnabActionUpgrade   = cnabAction("upgrade")
	cnabActionStatus    = cnabAction("status")
	cnabActionInspect   = cnabAction("inspect")
)

type cnabOperation struct {
	action       cnabAction
	installation string
}

func getCnabAction() (cnabAction, error) {
	action, ok := os.LookupEnv("CNAB_ACTION")
	if !ok {
		return "", errors.New("no CNAB action specified")
	}
	return cnabAction(action), nil
}

func getCnabOperation() (cnabOperation, error) {
	// CNAB_ACTION should always be set. but in future we want to have
	// claim-less actions. So we don't fail if no installation is set
	action, err := getCnabAction()
	if err != nil {
		return cnabOperation{}, err
	}
	return cnabOperation{
		action:       action,
		installation: os.Getenv("CNAB_INSTALLATION_NAME"),
	}, nil
}

func main() {
	op, err := getCnabOperation()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while parsing cnab operation: %s", err)
		os.Exit(1)
	}
	switch op.action {
	case cnabActionInstall:
		if err := install(op.installation); err != nil {
			fmt.Fprintf(os.Stderr, "Install failed: %s", err)
			os.Exit(1)
		}
	case cnabActionUpgrade:
		if err := install(op.installation); err != nil {
			fmt.Fprintf(os.Stderr, "Upgrade failed: %s", err)
			os.Exit(1)
		}
	case cnabActionUninstall:
		if err := uninstall(op.installation); err != nil {
			fmt.Fprintf(os.Stderr, "Uninstall failed: %s", err)
			os.Exit(1)
		}
	case cnabActionStatus:
		if err := status(op.installation); err != nil {
			fmt.Fprintf(os.Stderr, "Status failed: %s", err)
			os.Exit(1)
		}
	case cnabActionInspect:
		if err := inspect(); err != nil {
			fmt.Fprintf(os.Stderr, "Inspect failed: %s", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Action %q is not supported", op.action)
		os.Exit(1)
	}
}
