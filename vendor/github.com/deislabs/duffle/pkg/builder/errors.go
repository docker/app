package builder

import "errors"

var (
	// ErrDockerfileNotExist is returned when no Dockerfile exists during "duffle build."
	ErrDockerfileNotExist = errors.New("Dockerfile does not exist")
)
