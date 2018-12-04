package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/deis/duffle/pkg/action"
	"github.com/deis/duffle/pkg/bundle"
	"github.com/deis/duffle/pkg/claim"
	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/repo"
)

func newInstallCmd() *cobra.Command {
	const usage = `Installs a CNAB bundle.

This installs a CNAB bundle with a specific installation name. Once the install is complete,
this bundle can be referenced by installation name.

Example:
	$ duffle install my_release duffle/example:0.1.0
	$ duffle status my_release

Different drivers are available for executing the duffle invocation image. The following drivers
are built-in:

	- docker: run the Docker client. Works for OCI and Docker images
	- debug: fake a run of the invocation image, and print out what would have been sent

Some drivers have additional configuration that can be passed via environment variable.

	docker:
	  - VERBOSE: "true" turns on extra output

UNIX Example:
	$ VERBOSE=true duffle install -d docker my_release duffle/example:0.1.0

Windows Example:
	$ $env:VERBOSE = true
	$ duffle install -d docker my_release duffle/example:0.1.0

For unpublished CNAB bundles, you can also load the bundle.json directly:

	$ duffle install dev_bundle -f path/to/bundle.json


Verifying and --insecure:

  When the --insecure flag is passed, verification steps will not be performed. This means
  that Duffle will accept both unsigned (bundle.json) and signed (bundle.cnab) files, but
  will not perform any validation. The following table illustrates this:

	Bundle     Key known?    Flag            Result
	------     ----------    -----------     ------
	Signed     Known         None            Okay
	Signed     Known         --insecure      Okay
	Signed     Unknown       None            Verification error
	Signed     Unknown       --insecure      Okay
	Unsigned   N/A           None            Verification error
	Unsigned   N/A           --insecure      Okay
`
	var (
		installDriver    string
		credentialsFiles []string
		valuesFile       string
		bundleFile       string
		setParams        []string
		insecure         bool
		setFiles         []string

		installationName string
		bun              *bundle.Bundle
	)

	cmd := &cobra.Command{
		Use:   "install NAME BUNDLE",
		Short: "install a CNAB bundle",
		Long:  usage,
		RunE: func(cmd *cobra.Command, args []string) error {
			bundleFile, err := bundleFileOrArg2(args, bundleFile, cmd.OutOrStdout(), insecure)
			if err != nil {
				return err
			}
			installationName = args[0]

			// look in claims store for another claim with the same name
			_, err = claimStorage().Read(installationName)
			if err != claim.ErrClaimNotFound {
				return fmt.Errorf("a claim with the name %v already exists", installationName)
			}

			bun, err = loadBundle(bundleFile, insecure)
			if err != nil {
				return err
			}

			if err = bun.Validate(); err != nil {
				return err
			}

			driverImpl, err := prepareDriver(installDriver)
			if err != nil {
				return err
			}

			creds, err := loadCredentials(credentialsFiles, bun)
			if err != nil {
				return err
			}

			// Because this is an install, we create a new claim. For upgrades, we'd
			// load the claim based on installationName
			c, err := claim.New(installationName)
			if err != nil {
				return err
			}

			c.Bundle = bun
			c.Parameters, err = calculateParamValues(bun, valuesFile, setParams, setFiles)
			if err != nil {
				return err
			}

			inst := &action.Install{
				Driver: driverImpl,
			}
			fmt.Println("Executing install action...")
			err = inst.Run(c, creds, cmd.OutOrStdout())

			// Even if the action fails, we want to store a claim. This is because
			// we cannot know, based on a failure, whether or not any resources were
			// created. So we want to suggest that the user take investigative action.
			err2 := claimStorage().Store(*c)
			if err != nil {
				return fmt.Errorf("Install step failed: %v", err)
			}
			return err2
		},
	}

	flags := cmd.Flags()
	flags.BoolVarP(&insecure, "insecure", "k", false, "Do not verify the bundle (INSECURE)")
	flags.StringVarP(&installDriver, "driver", "d", "docker", "Specify a driver name")
	flags.StringVarP(&valuesFile, "parameters", "p", "", "Specify file containing parameters. Formats: toml, MORE SOON")
	flags.StringVarP(&bundleFile, "file", "f", "", "Bundle file to install")
	flags.StringArrayVarP(&credentialsFiles, "credentials", "c", []string{}, "Specify credentials to use inside the CNAB bundle. This can be a credentialset name or a path to a file.")
	flags.StringArrayVarP(&setParams, "set", "s", []string{}, "Set individual parameters as NAME=VALUE pairs")
	flags.StringArrayVarP(&setFiles, "set-file", "i", []string{}, "Set individual parameters from file content as NAME=SOURCE-PATH pairs")
	return cmd
}

func bundleFileOrArg2(args []string, bun string, w io.Writer, insecure bool) (string, error) {
	switch {
	case len(args) < 1:
		return "", errors.New("this command requires at least one argument: NAME (name for the installation). It also requires a BUNDLE (CNAB bundle name) or file (using -f)\nValid inputs:\n\t$ duffle install NAME BUNDLE\n\t$ duffle install NAME -f path-to-bundle.json")
	case len(args) == 2 && bun != "":
		return "", errors.New("please use either -f or specify a BUNDLE, but not both")
	case len(args) < 2 && bun == "":
		return "", errors.New("required arguments are NAME (name of the installation) and BUNDLE (CNAB bundle name) or file")
	case len(args) == 2:
		return getBundleFilepath(args[1], insecure)
	}
	return bun, nil
}

// optBundleFileOrArg2 optionally gets a bundle.
// Returning an empty string with no error is a possible outcome.
func optBundleFileOrArg2(args []string, bun string, w io.Writer, insecure bool) (string, error) {
	switch {
	case len(args) < 1:
		// No bundle provided
		return "", nil
	case len(args) == 2 && bun != "":
		return "", errors.New("please use either -f or specify a BUNDLE, but not both")
	case len(args) < 2 && bun == "":
		// No bundle provided
		return "", nil
	case len(args) == 2:
		return getBundleFilepath(args[1], insecure)
	}
	return bun, nil
}

func getBundleFilepath(bun string, insecure bool) (string, error) {
	home := home.Home(homePath())
	ref, err := getReference(bun)
	if err != nil {
		return "", fmt.Errorf("could not parse reference for %s: %v", bun, err)
	}

	// read the bundle reference from repositories.json
	index, err := repo.LoadIndex(home.Repositories())
	if err != nil {
		return "", fmt.Errorf("cannot open %s: %v", home.Repositories(), err)
	}

	digest, err := index.GetExactly(ref)
	if err != nil {
		return "", fmt.Errorf("could not find %s in %s: %v", ref.Name(), home.Repositories(), err)
	}
	return filepath.Join(home.Bundles(), digest), nil
}

// overrides parses the --set data and returns values that should override other params.
func overrides(overrides []string, paramDefs map[string]bundle.ParameterDefinition) (map[string]interface{}, error) {
	res := map[string]interface{}{}
	for _, p := range overrides {
		pair := strings.SplitN(p, "=", 2)
		if len(pair) != 2 {
			// For now, I guess we skip cases where someone does --set foo or --set foo=
			// We could set this to an explicit nil and then use it as a trigger to unset
			// a parameter in the file.
			continue
		}
		def, ok := paramDefs[pair[0]]
		if !ok {
			return res, fmt.Errorf("parameter %s not defined in bundle", pair[0])
		}

		if _, ok := res[pair[0]]; ok {
			return res, fmt.Errorf("parameter %q specified multiple times", pair[0])
		}

		var err error
		res[pair[0]], err = def.ConvertValue(pair[1])
		if err != nil {
			return res, fmt.Errorf("cannot use %s as value of %s: %s", pair[1], pair[0], err)
		}
	}
	return res, nil
}

func parseValues(file string) (map[string]interface{}, error) {
	v := viper.New()
	v.SetConfigFile(file)
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	return v.AllSettings(), nil
}

func calculateParamValues(bun *bundle.Bundle, valuesFile string, setParams, setFilePaths []string) (map[string]interface{}, error) {
	vals := map[string]interface{}{}
	if valuesFile != "" {
		var err error
		vals, err = parseValues(valuesFile)
		if err != nil {
			return vals, err
		}

	}
	overridden, err := overrides(setParams, bun.Parameters)
	if err != nil {
		return vals, err
	}
	for k, v := range overridden {
		vals[k] = v
	}

	// Now add files.
	for _, p := range setFilePaths {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			return vals, fmt.Errorf("malformed set-file parameter: %q (must be NAME=PATH)", p)
		}

		// Check that this is a known param
		if _, ok := bun.Parameters[parts[0]]; !ok {
			return vals, fmt.Errorf("bundle does not have a parameter named %q", parts[0])
		}

		if _, ok := overridden[parts[0]]; ok {
			return vals, fmt.Errorf("parameter %q specified multiple times", parts[0])
		}
		content, err := ioutil.ReadFile(parts[1])
		if err != nil {
			return vals, fmt.Errorf("could not read file %q: %s", parts[1], err)
		}
		vals[parts[0]] = string(content)
	}

	return bundle.ValuesOrDefaults(vals, bun)
}
