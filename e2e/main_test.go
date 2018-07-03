package e2e

import (
	"flag"
	"os"
	"testing"
)

var (
	e2ePath = flag.String("e2e-path", ".", "Set path to the e2e directory")
)

func TestMain(m *testing.M) {
	flag.Parse()
	if err := os.Chdir(*e2ePath); err != nil {
		panic(err)
	}
	m.Run()
}
