package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/deis/duffle/pkg/duffle/home"
	"github.com/deis/duffle/pkg/repo"
)

func newPushCmd(out io.Writer) *cobra.Command {
	const usage = `Pushes a CNAB bundle to a repository.`

	cmd := &cobra.Command{
		Hidden: true,
		Use:    "push NAME",
		Short:  "push a CNAB bundle to a repository",
		Long:   usage,
		Args:   cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			home := home.Home(homePath())
			bundleName := args[0]

			ref, err := getReference(bundleName)
			if err != nil {
				return fmt.Errorf("could not parse reference for %s: %v", bundleName, err)
			}

			// read the bundle reference from repositories.json
			index, err := repo.LoadIndex(home.Repositories())
			if err != nil {
				return fmt.Errorf("cannot open %s: %v", home.Repositories(), err)
			}

			digest, err := index.GetExactly(ref)
			if err != nil {
				return err
			}

			body, err := os.Open(filepath.Join(home.Bundles(), digest))
			if err != nil {
				return err
			}
			defer body.Close()

			url, err := repoURLFromReference(ref)
			if err != nil {
				return err
			}

			req, err := http.NewRequest("POST", url.String(), body)
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			fmt.Fprintf(out, "Successfully pushed %s\n", ref.String())
			return nil
		},
	}

	return cmd
}
