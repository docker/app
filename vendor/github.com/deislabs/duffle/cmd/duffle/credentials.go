package main

import (
	"io"

	"github.com/spf13/cobra"
)

const credentialDesc = `
Manages credential sets.

A credential set (credentialset) is a collection of credentialing information. It assigns a
name to place where a credential can be found. Credential sets are used to inject credentials
into an invocation image during operations such as 'duffle install' or 'duffle upgrade'.

Credential sets work by associating a local named credential with a credential requested by
a CNAB bundle. For example, if a CNAB bundle requires a credential named 'user_token',
a credential set can declare that 'user_token' is  fulfilled by loading the value
from the environment variable 'USER_TOKEN'.

Various command, such as 'duffle install', provide the '--credentials'/'-c' flag for pointing
to a credential set.

Duffle provides local credential set storage in the Duffle configuration directory. On a
UNIX-like system, these are stored in '$HOME/.duffle/credentials'. But credential sets are
just text files that map credential names to local sources.

A credential set can retrieve actual credentials from the following four sources:

	- a hard-coded value
	- an environment variable in the local environment
	- a file on the local file system
	- a command executed on the local system

Note that in all of these cases, the local system (the system running Duffle) is used as
the source of the credential.
`

func newCredentialsCmd(w io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "credentials",
		Short:   "manage credential sets",
		Long:    credentialDesc,
		Aliases: []string{"creds", "credential", "cred"},
	}

	cmd.AddCommand(
		newCredentialListCmd(w),
		newCredentialRemoveCmd(w),
		newCredentialAddCmd(w),
		newCredentialShowCmd(w),
		newCredentialGenerateCmd(w),
		newCredentialEditCmd(w),
	)

	return cmd
}
