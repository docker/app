package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/repo"
)

// ReferenceToDigestList is a list of bundle references.
// Implements a sorter on Name.
type ReferenceToDigestList []*ReferenceToDigest

// Len returns the length.
func (bl ReferenceToDigestList) Len() int { return len(bl) }

// Swap swaps the position of two items in the versions slice.
func (bl ReferenceToDigestList) Swap(i, j int) { bl[i], bl[j] = bl[j], bl[i] }

// Less returns true if the version of entry a is less than the version of entry b.
func (bl ReferenceToDigestList) Less(a, b int) bool {
	return strings.Compare(reference.FamiliarString(bl[a].ref), reference.FamiliarString(bl[a].ref)) < 1
}

// ReferenceToDigest is a reference to a repository with a name, tag and digest.
type ReferenceToDigest struct {
	ref    reference.NamedTagged
	digest string
}

// Name returns the full name.
func (n *ReferenceToDigest) String() string {
	return reference.FamiliarString(n.ref)
}

// Name returns the name.
func (n *ReferenceToDigest) Name() string {
	return reference.FamiliarName(n.ref)
}

// Tag returns the tag.
func (n *ReferenceToDigest) Tag() string {
	return n.ref.Tag()
}

// Digest returns the digest.
func (n *ReferenceToDigest) Digest() string {
	return n.digest
}

func newBundleListCmd(w io.Writer) *cobra.Command {
	var long bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "lists bundles pulled or built and stored locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			home := home.Home(homePath())
			references, err := searchLocal(home)
			if err != nil {
				return err
			}
			sort.Sort(references)
			if long {
				table := uitable.New()
				table.AddRow("NAME", "VERSION", "DIGEST")
				for _, ref := range references {
					table.AddRow(ref.Name(), ref.Tag(), ref.Digest())
				}
				fmt.Fprintln(w, table)
				return nil
			}

			for _, ref := range references {
				fmt.Println(ref)
			}

			return nil
		},
	}
	cmd.Flags().BoolVarP(&long, "long", "l", false, "output longer listing format")

	return cmd
}

func searchLocal(home home.Home) (ReferenceToDigestList, error) {
	references := ReferenceToDigestList{}

	index, err := repo.LoadIndex(home.Repositories())
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %v", home.Repositories(), err)
	}

	for repo, tagList := range index {
		named, err := reference.ParseNormalizedNamed(repo)
		if err != nil {
			return nil, err
		}
		for tag, digest := range tagList {
			tagged, err := reference.WithTag(named, tag)
			if err != nil {
				return nil, err
			}
			references = append(references, &ReferenceToDigest{
				ref:    tagged,
				digest: digest,
			})
		}
	}

	return references, nil
}
