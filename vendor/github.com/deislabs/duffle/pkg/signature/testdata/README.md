# Instructions for Working with Keys

If you are using GnuPG, this document explains how to build keyrings.

## Building a New Keybox and Keyring

```
# Easy
$ gpg --no-default-keyring --keyring $(pwd)/keyring.kbx --quick-generate-key foo@example.com
# Complete (recommended)
$ gpg --no-default-keyring --keyring $(pwd)/keyring.kbx --full-generate-key
```

Export a binary keyring to an ASCII-armored keyring:

```
$ gpg --no-default-keyring --keyring $(pwd)/keyring.kbx --export-secret-keys --output ./keyring.gpg
```

Now the `keyring.gpg` is the file to import.

To find out various key ids, this command is the easiest:

```
$ gpg --keyring ./keyring.gpg --no-default-keyring --list-keys --list-options=show-uid --with-colons
```