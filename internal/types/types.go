package types

import (
	"path/filepath"

	"github.com/docker/app/internal"
)

// App represents an app (extracted or not)
type App struct {
	Path          string
	OriginalPath  string
	ComposeFiles  []string
	SettingsFiles []string
	MetadataFile  string
	Cleanup       func()
}

// NewApp creates a new docker app with the specified path and struct modifiers
func NewApp(path string, ops ...func(*App)) App {
	app := &App{
		Path:          path,
		ComposeFiles:  []string{filepath.Join(path, internal.ComposeFileName)},
		SettingsFiles: []string{filepath.Join(path, internal.SettingsFileName)},
		MetadataFile:  filepath.Join(path, internal.MetadataFileName),
	}

	for _, op := range ops {
		op(app)
	}

	return *app
}

// WithOriginalPath sets the original path of the app
func WithOriginalPath(path string) func(*App) {
	return func(app *App) {
		app.OriginalPath = path
	}
}

// WithCleanup sets the cleanup function of the app
func WithCleanup(f func()) func(*App) {
	return func(app *App) {
		app.Cleanup = f
	}
}

// WithSettingsFiles adds the specified settings files of the app
func WithSettingsFiles(files ...string) func(*App) {
	return func(app *App) {
		app.SettingsFiles = append(app.SettingsFiles, files...)
	}
}

// WithComposeFiles adds the specified compose files of the app
func WithComposeFiles(files ...string) func(*App) {
	return func(app *App) {
		app.ComposeFiles = append(app.ComposeFiles, files...)
	}
}
