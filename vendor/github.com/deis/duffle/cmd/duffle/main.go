package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/deis/duffle/pkg/signature"

	"github.com/spf13/cobra"

	"github.com/deis/duffle/pkg/bundle"
	"github.com/deis/duffle/pkg/claim"
	"github.com/deis/duffle/pkg/credentials"
	"github.com/deis/duffle/pkg/driver"
	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/utils/crud"
)

var (
	// duffleHome depicts the home directory where all duffle config is stored.
	duffleHome string
	rootCmd    *cobra.Command
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				must(err)
			}
		}
	}()
	rootCmd = newRootCmd(nil)
	must(rootCmd.Execute())
}

func homePath() string {
	return os.ExpandEnv(duffleHome)
}

func defaultDuffleHome() string {
	if home := os.Getenv(home.HomeEnvVar); home != "" {
		return home
	}

	homeEnvPath := os.Getenv("HOME")
	if homeEnvPath == "" && runtime.GOOS == "windows" {
		homeEnvPath = os.Getenv("USERPROFILE")
	}

	return filepath.Join(homeEnvPath, ".duffle")
}

// claimStorage returns a claim store for accessing claims.
func claimStorage() claim.Store {
	h := home.Home(homePath())
	return claim.NewClaimStore(crud.NewFileSystemStore(h.Claims(), "json"))
}

// loadCredentials loads a set of credentials from HOME.
func loadCredentials(files []string, b *bundle.Bundle) (map[string]string, error) {
	creds := map[string]string{}
	if len(files) == 0 {
		return creds, credentials.Validate(creds, b.Credentials)
	}

	// The strategy here is "last one wins". We loop through each credential file and
	// calculate its credentials. Then we insert them into the creds map in the order
	// in which they were supplied on the CLI.
	for _, file := range files {
		if !isPathy(file) {
			file = filepath.Join(home.Home(homePath()).Credentials(), file+".yaml")
		}
		cset, err := credentials.Load(file)
		if err != nil {
			return creds, err
		}
		res, err := cset.Resolve()
		if err != nil {
			return res, err
		}

		for k, v := range res {
			creds[k] = v
		}
	}
	return creds, credentials.Validate(creds, b.Credentials)
}

// loadVerifyingKeyRings loads all of the keys that can be used for verifying.
//
// This includes all the keys in the public key file and in the secret key file.
func loadVerifyingKeyRings(homedir string) (*signature.KeyRing, error) {
	hp := home.Home(homedir)
	return signature.LoadKeyRings(hp.PublicKeyRing(), hp.SecretKeyRing())
}

// isPathy checks to see if a name looks like a path.
func isPathy(name string) bool {
	return strings.Contains(name, string(filepath.Separator))
}

func must(err error) {
	if err != nil {
		os.Exit(1)
	}
}

// prepareDriver prepares a driver per the user's request.
func prepareDriver(driverName string) (driver.Driver, error) {
	driverImpl, err := driver.Lookup(driverName)
	if err != nil {
		return driverImpl, err
	}

	// Load any driver-specific config out of the environment.
	if configurable, ok := driverImpl.(driver.Configurable); ok {
		driverCfg := map[string]string{}
		for env := range configurable.Config() {
			driverCfg[env] = os.Getenv(env)
		}
		configurable.SetConfig(driverCfg)
	}

	return driverImpl, err
}
