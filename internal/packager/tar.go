package packager

import (
	"archive/tar"
	"io"

	"github.com/pkg/errors"
)

type tarOperation func(*tar.Reader, *tar.Header) error
type tarStatusOperation func(*tar.Reader, *tar.Header) (bool, error)

func handleTar(file io.Reader, handler tarOperation) error {
	tarReader := tar.NewReader(file)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "error reading from tar header")
		}
		err = handler(tarReader, header)
		if err != nil {
			return err
		}
	}
	return nil
}

func handleTarWithStatus(file io.Reader, handler tarStatusOperation) (bool, error) {
	status := false
	tarReader := tar.NewReader(file)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status, errors.Wrap(err, "error reading from tar header")
		}
		tmpStatus, err := handler(tarReader, header)
		if err != nil {
			return status, err
		}
		status = status || tmpStatus
	}
	return status, nil
}
