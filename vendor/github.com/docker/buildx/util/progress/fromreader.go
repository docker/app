package progress

import (
	"io"
	"io/ioutil"
	"time"

	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/identity"
	"github.com/opencontainers/go-digest"
)

func FromReader(w Writer, name string, rc io.ReadCloser) {
	status := w.Status()
	dgst := digest.FromBytes([]byte(identity.NewID()))
	tm := time.Now()

	vtx := client.Vertex{
		Digest:  dgst,
		Name:    name,
		Started: &tm,
	}

	status <- &client.SolveStatus{
		Vertexes: []*client.Vertex{&vtx},
	}

	_, err := io.Copy(ioutil.Discard, rc)

	tm2 := time.Now()
	vtx2 := vtx
	vtx2.Completed = &tm2
	if err != nil {
		vtx2.Error = err.Error()
	}
	status <- &client.SolveStatus{
		Vertexes: []*client.Vertex{&vtx2},
	}
	close(status)
}
