# Signing and Verifying in Duffle

CNAB describes a mechanism for signing and verifying bundles. Duffle supports the CNAB spec. This guide explains how Duffle's signing and verifying works.

## The Basic Idea

CNAB uses what is called a "web of trust" model for ensuring that bundles are trustworthy. In the "web of trust" model, you decide whom you trust. This is not established by a central authority (which is the way SSL works). Instead, you decide based on whether you believe the other person is trustworthy. You might decide to only trust people you personally know. Or perhaps you trust not only those people, but the people they suggest are trustworthy. Or maybe you only want to trust large corporations. The point is, in the web of trust model _it's up to you_.

The web of trust model works like this:

- You have (or create) a _key pair_. This has two keys: a _signing key_ which you keep private, and a _verifying key_ that you can share.
    - With a singing key, you can _sign a bundle_, effectively stating, "I, [insert your name], hereby claim that I trust this bundle."
    - You can go ahead and distribute your _verifying key_ to other people. There's nothing private about this key. It's primary use is for other people to be able to use it to verify that you indeed signed something.
- When you distribute your signed bundle, other people can _use your verifying key to check your signature_.
    - Your friends can then _import your verifying key_.
    - If the bundle was signed with your private key, Duffle will let your friends know that the bundle is trusted.
    - If it fails to verify, Duffle will not let your friends install the bundle. It will tell them that either they are missing the right key or someone forged a signature.
- You will want to keep your signing key secret and safe, and use it consistently when signing your bundles. So if you have two computers that you use for creating bundles, that _usually_ means you want both to have the same signing key.

That's the basic idea of the way Duffle does web of trust. There's more to web of trust, though. You can, for example, sign other people's public keys in order to say, "these are people I trust." In this case, you can share those signed keys with your friends, who can say, "well, if [insert your name] trusts them, so will I." Duffle does not yet have a lot of built-in support for this, but there are other tools that you can use with Duffle (like Gnu Privacy Guard, or `gpg`) to do that part. We also really like [Keybase](https://keybase.io), which provides a really cool way of building a trust model for your OpenPGP keys.

## Cryptographic Batteries Included

Now we can get started with the practical stuff. Duffle tries to make this as painless as possible for you.

The first time you run Duffle, it will create a a signing key for you. It does this automatically, and it places the key in `$HOME/.duffle/secret.ring`. You can look at your secret keyring like this:

```console
$ duffle key list --signing
Test One (Signer) <test1@example.com>
```

As you can see, the keyring above has one key. My personal keyring has 4 (a personal, a work key, and a few for testing).

Since you already have a signing key, whenever you create new bundles, they will automatically be signed with this key:

```console
$ duffle create mybundle
$ # work on your bundle
$ duffle build
```

The `duffle build` command will automatically sign your bundle with the first key it finds in your secret ring. If you have multiple keys, you can use the `--user`/`-u` flag to choose one:

```
$ duffle build -u test1@example.com
```

## Verifying Bundles

Just as with signing bundles, Duffle does its best to automatically verify a bundle as well. Above, we used `duffle build` to create a bundle. Since we signed it with our own key, we can also easily verify it. And this happens automatically when we do things like `duffle install` or `duffle upgrade`:

```console
$ duffle install myrelease mybundle
```

However, when it comes to working with bundles that were created by other people you trust, you need to first get their public key.

Say they give you a key named `friend.key` (the extension, by the way, makes no difference to Duffle). You can import it into your _verifying keys_ like this:

```console
$ duffle key import friend.key
```

Then you can verify that it imported by listing the _verifying keys_ that are in your keyring:

```console
$ duffle key list -l
NAME                                    TYPE    FINGERPRINT
Extra Key (Signer) <extra1@example.com> signing D113 40F5 336 A06C BF0 7F14 F8A4 26C5 52CB 806
Test One (Signer) <test1@example.com>   signing FAB9 6672 1CFB 1C25 164D BB4E 281 57BA FDA1 54FA
friend@example.com <friend@example.com> signing 5D76 712C E625 98A8 27A2 7E28 9B79 91DD 4037 8340
```

In the example above, we have three keys that can be used for verifying bundles.

## Sharing Your Verifying Key with Others

When your signing key was generated, Duffle also generated a verifying key for you to share with your friends. You can _export_ this key to a file, and then share that file:

```console
$ duffle key export my_signer.key
```

That will export _just the first key_ that you generated. If you want to export particular keys, you can use the `--user`/`-u` flag:

```console
$ duffle key export my_signer.key -u test1@example.com
```

Now, any of your friends can use `duffle key import` to add your key to their keyring.

## But I Want to Use My Existing Key!

Say you already have a key (from, perhaps, Keybase or GnuPG). In most cases, you can use that key with Duffle. There are two ways of doing this:

- You can use `duffle key import --secret mykey.gpg` and import it to the existing keyring. However, this will place the key at the _end_ of your keyring, and you will need to use the `--user`/`-u` flag whenever signing.
- You can remove your existing `secret.ring` and then run `duffle init -s mykey.gpg` or run `duffle key import --secret mykey.gpg`. Either of these will create a new keyring for you.

As always, you can check on your keyring by using `duffle key list --secret` (or see your verifying keys with `duffle key list`).

## Manually Signing and Verifying Bundles

Most of the time Duffle will deal with signing and verifying for you. But it also provides a few extra tools you can use to sign and verify on your own. These are useful if you are using other tools to build or install your CNAB bundles, but just need to do the signing or verifying yourself.

Verifying a bundle is done with `duffle bundle verify`:

```
$ duffle bundle verify -f ./bundle.cnab
Signed by "Extra Key (Signer) <extra1@example.com>" (D113 40F5 336 A06C BF0 7F14 F8A4 26C5 52CB 806)
```

A verification works like this:

- it reads the signature on a `bundle.cnab` file
- it tests the signature against every key in your keyring until it finds one that works
- it then returns the key whose verification passed
- and it exits with code 0

If it cannot find a key that will verify the bundle, it will print an error message and exit with the error code 1.

You can sign a bundle like this:

```
$ duffle bundle sign -f bundle.json -o bundle.cnab
```

The above will use your default signing key to sign the file `bundle.json` and produce the file `bundle.cnab`. As usual, you can use `--user`/`-u` to specify a particular user ID.

## Conclusion

The purpose of this guide has been to explain how signing and verifying work in Duffle, and which commands are available to you. If you are deeply interested in the theory and implementation behind OpenPGP-style signing, you may want to read the [OpenPGP spec](https://tools.ietf.org/html/rfc4880).