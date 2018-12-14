package crud

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// ErrFileDoesNotExist represents when file path is not found on file system
var ErrFileDoesNotExist = errors.New(`File does not exist`)

// NewFileSystemStore creates a Store backed by a file system directory.
// Each key is represented by a file in that directory.
func NewFileSystemStore(baseDirectory string, fileExtension string) Store {
	return fileSystemStore{
		baseDirectory: baseDirectory,
		fileExtension: fileExtension,
	}
}

type fileSystemStore struct {
	baseDirectory string
	fileExtension string
}

func (s fileSystemStore) List() ([]string, error) {
	if err := s.ensure(); err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(s.baseDirectory)
	if err != nil {
		return []string{}, err
	}

	return names(s.storageFiles(files)), nil
}

func (s fileSystemStore) Store(name string, data []byte) error {
	filename, err := s.fullyQualifiedName(name)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, os.ModePerm)
}

func (s fileSystemStore) Read(name string) ([]byte, error) {
	filename, err := s.fullyQualifiedName(name)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, ErrFileDoesNotExist
	}

	return ioutil.ReadFile(filename)
}

func (s fileSystemStore) Delete(name string) error {
	filename, err := s.fullyQualifiedName(name)
	if err != nil {
		return err
	}
	return os.Remove(filename)
}

func (s fileSystemStore) fileNameOf(name string) string {
	return filepath.Join(s.baseDirectory, fmt.Sprintf("%s.%s", name, s.fileExtension))
}

func (s fileSystemStore) fullyQualifiedName(name string) (string, error) {
	if err := s.ensure(); err != nil {
		return "", err
	}
	return s.fileNameOf(name), nil
}

func (s fileSystemStore) ensure() error {
	fi, err := os.Stat(s.baseDirectory)
	if err == nil {
		if fi.IsDir() {
			return nil
		}
		return errors.New("Storage directory name exists, but is not a directory")
	}
	return os.MkdirAll(s.baseDirectory, os.ModePerm)
}

func (s fileSystemStore) storageFiles(files []os.FileInfo) []os.FileInfo {
	result := make([]os.FileInfo, 0)
	ext := "." + s.fileExtension
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ext {
			result = append(result, file)
		}
	}
	return result
}

func names(files []os.FileInfo) []string {
	result := make([]string, 0)
	for _, file := range files {
		result = append(result, name(file.Name()))
	}
	return result
}

func name(filename string) string {
	return strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
}
