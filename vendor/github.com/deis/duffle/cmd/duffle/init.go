package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/AlecAivazis/survey.v1"

	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/ohai"
	"github.com/deis/duffle/pkg/signature"

	"github.com/spf13/cobra"
)

const (
	initDesc = `
Explicitly control the creation of the Duffle environment.

Normally, Duffle initializes itself. But on occasion, you may wish to customize Duffle's initialization,
passing your own keys or testing to see what directories will be created. This command is provided for
such a reason.

This command will create a subdirectory in your home directory, and use that directory for storing
configuration, preferences, and persistent data. Duffle uses OpenPGP-style keys for signing and
verification. If you do not provide a secret key to import, the init phase will generate a keyring for
you, and create a signing key.

During initialization, you may use '--public-keys' to import a keyring of public keys. These keys will
then be used by other commands (such as 'duffle install') to verify the integrity of a package. If
you do not supply keys during initialization, you will need to provide them later. WARNING: You should
not import private keys with the '--public-keys' flag, or they may be placed in your public keyring.
`
)

type initCmd struct {
	dryRun     bool
	keyFile    string
	username   string
	w          io.Writer
	pubkeyFile string
	verbose    bool
}

func newInitCmd(w io.Writer) *cobra.Command {
	i := &initCmd{w: w, verbose: true}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "set up local environment to work with duffle",
		Long:  initDesc,
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if i.keyFile != "" && i.username != "" {
				fmt.Fprintln(os.Stderr, "WARNING: 'user' and 'signing-key' were both provided. Ignoring 'user'.")
			}
			return i.run()
		},
	}

	f := cmd.Flags()
	f.BoolVar(&i.dryRun, "dry-run", false, "go through all the steps without actually installing anything")
	f.StringVarP(&i.keyFile, "signing-key", "k", "", "Armored OpenPGP key to be used for signing. If not specified, one will be generated for you")
	f.StringVarP(&i.pubkeyFile, "public-keys", "p", "", "Armored OpenPGP key containing trusted public keys. If not specified, no public keys will be trusted by default")
	f.StringVarP(&i.username, "user", "u", "", "User identity for the OpenPGP key. The format is 'NAME (OPTIONAL COMMENT) <EMAIL@ADDRESS>'.")

	return cmd
}

// autoInit is called by the root command for all calls except init.
func autoInit(w io.Writer, verbose bool) error {
	i := initCmd{
		w:       w,
		verbose: verbose,
	}
	return i.run()
}

func (i *initCmd) run() error {
	home := home.Home(homePath())
	dirs := []string{
		home.String(),
		home.Bundles(),
		home.Logs(),
		home.Plugins(),
		home.Claims(),
		home.Credentials(),
	}

	files := []string{
		home.Repositories(),
	}

	if i.verbose {
		ohai.Fohailn(i.w, "The following new directories will be created:")
		fmt.Fprintln(i.w, strings.Join(dirs, "\n"))
	}

	if !i.dryRun {
		if err := ensureDirectories(dirs); err != nil {
			return err
		}
	}

	if i.verbose {
		ohai.Fohailn(i.w, "The following new files will be created:")
		fmt.Fprintln(i.w, strings.Join(files, "\n"))
	}

	if !i.dryRun {
		if err := ensureFiles(files); err != nil {
			return err
		}
	}

	pkr, err := i.loadOrCreateSecretKeyRing(home.SecretKeyRing())
	if err != nil {
		return err
	}
	_, err = i.loadOrCreatePublicKeyRing(home.PublicKeyRing(), pkr)
	return err
}

func ensureDirectories(dirs []string) error {
	for _, dir := range dirs {
		if fi, err := os.Stat(dir); err != nil {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("Could not create %s: %s", dir, err)
			}
		} else if !fi.IsDir() {
			return fmt.Errorf("%s must be a directory", dir)
		}
	}
	return nil
}

func ensureFiles(files []string) error {
	for _, name := range files {
		f, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		f.Close()
	}
	return nil
}

// This loads a keyring from disk. If no keyring already exists, this will create a new
// keyring, add a new default identity, and then write that keyring to disk.
//
// Regardless of the path, a *signature.KeyRing will be returned.
func (i *initCmd) loadOrCreateSecretKeyRing(dest string) (*signature.KeyRing, error) {
	if _, err := os.Stat(dest); err == nil {
		// Since this is non-mutating, we can do this in a dry-run.
		return signature.LoadKeyRing(dest)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	fmt.Fprintf(i.w, "==> Generating a new secret keyring at %s\n", dest)

	// We could probably move the dry-run to just before the `ring.Save`. Not sure
	// what that accomplishes, though.
	if i.dryRun {
		return &signature.KeyRing{}, nil
	}

	ring := signature.CreateKeyRing(passwordFetcher)
	if i.keyFile != "" {
		key, err := os.Open(i.keyFile)
		if err != nil {
			return ring, err
		}
		err = ring.Add(key, true)
		key.Close()
		if err != nil {
			return ring, err
		}

		if all := ring.PrivateKeys(); len(all) == 0 {
			// If we have no private keys, this is probably an error condition, since
			// signing will be broken.
			return ring, errors.New("no private keys were found in the key file")
		}

		for _, k := range ring.PrivateKeys() {
			i.printUserID(k)
		}
	} else {
		var user signature.UserID
		if i.username != "" {
			var err error
			user, err = signature.ParseUserID(i.username)
			if err != nil {
				return ring, err
			}
		} else {
			user = defaultUserID()

		}
		// Generate the key
		fmt.Fprintf(i.w, "==> Generating a new signing key with ID %s\n", user.String())
		k, err := signature.CreateKey(user)
		if err != nil {
			return ring, err
		}
		ring.AddKey(k)
	}
	err := ring.SavePrivate(dest, false)
	if err != nil {
		return ring, err
	}

	return ring, os.Chmod(dest, 0600)

}

// loadOrCreatePublicKeyRing creates a ring of public keys.
// If the privateKeys are passed in, the public keys for each is then saved in the public keyring.
// This is useful if you need to verify things that were signed via one of the private keys on
// the secret ring.
func (i *initCmd) loadOrCreatePublicKeyRing(dest string, privateKeys *signature.KeyRing) (*signature.KeyRing, error) {
	if _, err := os.Stat(dest); err == nil {
		// Since this is non-mutating, we can do this in a dry-run.
		return signature.LoadKeyRing(dest)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	fmt.Fprintf(i.w, "==> Generating a new public keyring at %s\n", dest)

	// We could probably move the dry-run to just before the `ring.Save`. Not sure
	// what that accomplishes, though.
	if i.dryRun {
		return &signature.KeyRing{}, nil
	}
	ring := signature.CreateKeyRing(passwordFetcher)

	if i.pubkeyFile != "" {
		keys, err := os.Open(i.pubkeyFile)
		if err != nil {
			return ring, err
		}
		err = ring.Add(keys, true)
		keys.Close()
		if err != nil {
			return ring, err
		}
	}

	for _, k := range ring.Keys() {
		i.printUserID(k)
	}

	return ring, ring.SavePublic(dest, false)
}

func (i *initCmd) printUserID(k *signature.Key) {
	uid, err := k.UserID()
	if err != nil {
		fmt.Fprintln(i.w, "==> Importing anonymous key")
		return
	}
	fmt.Fprintf(i.w, "==> Importing %q\n", uid)
}

// passwordFetcher is a simple prompt-based no-echo password input.
func passwordFetcher(prompt string) ([]byte, error) {
	var pw string
	err := survey.AskOne(&survey.Password{
		Message: fmt.Sprintf("Passphrase for key %q >  ", prompt),
		Help:    "Unlock a passphrase-protected key",
	}, &pw, nil)
	return []byte(pw), err
}
