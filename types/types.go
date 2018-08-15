package types

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/app/internal"
)

// SingleFileSeparator is the separator used in single-file app
const SingleFileSeparator = "\n---\n"

// App represents an app
type App struct {
	Name    string
	Path    string
	Cleanup func()

	composesContent [][]byte
	settingsContent [][]byte
	metadataContent []byte
}

// Composes returns compose files content
func (a *App) Composes() [][]byte {
	return a.composesContent
}

// Settings returns setting files content
func (a *App) Settings() [][]byte {
	return a.settingsContent
}

// Metadata returns metadata file content
func (a *App) Metadata() []byte {
	return a.metadataContent
}

func noop() {}

// NewApp creates a new docker app with the specified path and struct modifiers
func NewApp(path string, ops ...func(*App) error) (*App, error) {
	app := &App{
		Name:    path,
		Path:    path,
		Cleanup: noop,

		composesContent: [][]byte{},
		settingsContent: [][]byte{},
		metadataContent: []byte{},
	}

	for _, op := range ops {
		if err := op(app); err != nil {
			return nil, err
		}
	}

	return app, nil
}

// NewAppFromDefaultFiles creates a new docker app using the default files in the specified path.
// If one of those file doesn't exists, it will error out.
func NewAppFromDefaultFiles(path string, ops ...func(*App) error) (*App, error) {
	appOps := append([]func(*App) error{
		MetadataFile(filepath.Join(path, internal.MetadataFileName)),
		WithComposeFiles(filepath.Join(path, internal.ComposeFileName)),
		WithSettingsFiles(filepath.Join(path, internal.SettingsFileName)),
	}, ops...)
	return NewApp(path, appOps...)
}

// WithName sets the application name
func WithName(name string) func(*App) error {
	return func(app *App) error {
		app.Name = name
		return nil
	}
}

// WithPath sets the original path of the app
func WithPath(path string) func(*App) error {
	return func(app *App) error {
		app.Path = path
		return nil
	}
}

// WithCleanup sets the cleanup function of the app
func WithCleanup(f func()) func(*App) error {
	return func(app *App) error {
		app.Cleanup = f
		return nil
	}
}

// WithSettingsFiles adds the specified settings files to the app
func WithSettingsFiles(files ...string) func(*App) error {
	return func(app *App) error {
		parsedFiles := []string{}
		for _, path := range files {
			psf := filepath.Join(app.Name, path)
			if _, err := os.Stat(psf); err == nil {
				parsedFiles = append(parsedFiles, psf)
			} else {
				parsedFiles = append(parsedFiles, path)
			}
		}
		settingsContent, err := readFiles(parsedFiles...)
		if err != nil {
			return err
		}
		app.settingsContent = append(app.settingsContent, settingsContent...)
		return nil
	}
}

// WithSettings adds the specified settings readers to the app
func WithSettings(readers ...io.Reader) func(*App) error {
	return func(app *App) error {
		settingsContent, err := readReaders(readers...)
		if err != nil {
			return err
		}
		app.settingsContent = append(app.settingsContent, settingsContent...)
		return nil
	}
}

// MetadataFile adds the specified metadata file to the app
func MetadataFile(file string) func(*App) error {
	return func(app *App) error {
		d, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		app.metadataContent = d
		return nil
	}
}

// Metadata adds the specified metadata reader to the app
func Metadata(r io.Reader) func(*App) error {
	return func(app *App) error {
		d, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		app.metadataContent = d
		return nil
	}
}

// WithComposeFiles adds the specified compose files to the app
func WithComposeFiles(files ...string) func(*App) error {
	return func(app *App) error {
		composesContent, err := readFiles(files...)
		if err != nil {
			return err
		}
		app.composesContent = append(app.composesContent, composesContent...)
		return nil
	}
}

// WithComposes adds the specified compose readers to the app
func WithComposes(readers ...io.Reader) func(*App) error {
	return func(app *App) error {
		composesContent, err := readReaders(readers...)
		if err != nil {
			return err
		}
		app.composesContent = append(app.composesContent, composesContent...)
		return nil
	}
}

func readReaders(readers ...io.Reader) ([][]byte, error) {
	content := make([][]byte, len(readers))
	var errs []string
	for i, r := range readers {
		d, err := ioutil.ReadAll(r)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		content[i] = d
	}
	return content, newErrGroup(errs)
}

func readFiles(files ...string) ([][]byte, error) {
	content := make([][]byte, len(files))
	var errs []string
	for i, file := range files {
		d, err := ioutil.ReadFile(file)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		content[i] = d
	}
	return content, newErrGroup(errs)
}

func newErrGroup(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "\n"))
}
