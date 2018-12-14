package main

import (
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/deislabs/duffle/pkg/signature"
)

// defaultUserID returns the default user name.
func defaultUserID() signature.UserID {
	// TODO: I am not sure how reliable this is on Windows.
	domain, err := os.Hostname()
	if err != nil {
		domain = "localhost.localdomain"
	}
	var name, username string

	if account, err := user.Current(); err != nil {
		name = "user"
	} else {
		name = account.Name
		username = account.Username
	}
	// on Windows, account name are prefixed with '<machinename>\' which makes the generated email invalid
	// and makes the key user identity parser fail
	if ix := strings.Index(username, "\\"); ix != -1 {
		username = username[ix+1:]
	}

	email := fmt.Sprintf("%s@%s", username, domain)
	return signature.UserID{
		Name:  name,
		Email: email,
	}
}
