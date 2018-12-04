package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/repo"

	"github.com/Masterminds/semver"
	"github.com/docker/distribution/reference"
	"github.com/spf13/cobra"
)

const bundleRemoveDesc = `Remove a bundle from the local storage.

This removes a bundle from the local storage so that it will no longer be locally
available. Bundles can be rebuilt with 'duffle build'.

If a SemVer range is provided with '--version'/'-r' then only releases that match
that range will be removed.
`

func newBundleRemoveCmd(w io.Writer) *cobra.Command {
	var versions string
	cmd := &cobra.Command{
		Use:     "remove BUNDLE",
		Aliases: []string{"rm"},
		Short:   "remove a bundle from the local storage",
		Long:    bundleRemoveDesc,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			named, err := reference.ParseNormalizedNamed(args[0])
			if err != nil {
				return err
			}
			tagged, _ := named.(reference.NamedTagged)
			if tagged != nil {
				if versions != "" {
					return errors.New("cannot set --version flag when bundle name has a tag")
				}
			}

			h := home.Home(homePath())
			index, err := repo.LoadIndex(h.Repositories())
			if err != nil {
				return err
			}

			vers, ok := index.GetVersions(named)
			if !ok {
				fmt.Fprintf(w, "Bundle %q not found. Nothing deleted.", args[0])
				return nil
			}

			// If versions is set, we short circuit and only delete specific versions.
			if versions != "" {
				fmt.Fprintln(w, "Only deleting versions")
				matcher, err := semver.NewConstraint(versions)
				if err != nil {
					return err
				}
				deletions := map[string]string{}
				for ver, sha := range vers {
					sv, err := semver.NewVersion(ver)
					if err != nil {
						fmt.Fprintf(w, "WARNING: %q is not a semantic version", ver)
					}
					if ok, _ := matcher.Validate(sv); ok {
						fmt.Fprintf(w, "Version %s matches constraint %q\n", ver, versions)
						deletions[ver] = sha
						versioned, err := reference.WithTag(named, ver)
						if err != nil {
							return err
						}
						index.DeleteVersion(versioned)
						// If there are no more versions, remove the entire entry.
						if vers, ok := index.GetVersions(named); ok && len(vers) == 0 {
							index.DeleteAll(named)
						}

					}
				}
				// We can skip writing the file if there is nothing to delete.
				if len(deletions) == 0 {
					return nil
				}
				if err := index.WriteFile(h.Repositories(), 0644); err != nil {
					return err
				}
				deleteBundleVersions(deletions, index, h, w)
				return nil
			}
			if tagged != nil {
				sha, ok := vers[tagged.Tag()]
				if !ok {
					return fmt.Errorf("version %q not found", tagged.Tag())
				}
				index.DeleteVersion(tagged)
				if err := index.WriteFile(h.Repositories(), 0644); err != nil {
					return err
				}
				if !isShaReferenced(index, sha) {
					fpath := filepath.Join(h.Bundles(), sha)
					if err := os.Remove(fpath); err != nil {
						fmt.Fprintf(w, "WARNING: could not delete stake record %q", fpath)
					}
				}
				return nil
			}

			// If no version was specified, delete entire record
			if !index.DeleteAll(named) {
				fmt.Fprintf(w, "Bundle %q not found. Nothing deleted.", named)
				return nil
			}
			if err := index.WriteFile(h.Repositories(), 0644); err != nil {
				return err
			}

			deleteBundleVersions(vers, index, h, w)
			return nil
		},
	}
	cmd.Flags().StringVarP(&versions, "version", "r", "", "A version or SemVer2 version range")

	return cmd
}

func isShaReferenced(index repo.Index, sha string) bool {
	for _, vs := range index {
		for _, otherSha := range vs {
			if otherSha == sha {
				return true
			}
		}
	}
	return false
}

// deleteBundleVersions removes the given SHAs from bundle storage
//
// It warns, but does not fail, if a given SHA is not found.
func deleteBundleVersions(vers map[string]string, index repo.Index, h home.Home, w io.Writer) {
	for _, sha := range vers {
		if !isShaReferenced(index, sha) {
			fpath := filepath.Join(h.Bundles(), sha)
			if err := os.Remove(fpath); err != nil {
				fmt.Fprintf(w, "WARNING: could not delete stake record %q", fpath)
			}
		}
	}
}
