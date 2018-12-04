package builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/semver"

	"github.com/deis/duffle/pkg/bundle"
	"github.com/deis/duffle/pkg/duffle/manifest"
)

// Builder defines how to interact with a bundle builder
type Builder struct {
	ID      string
	LogsDir string
	// If this is true, versions will contain build metadata
	// Example:
	//   0.1.2+2c3c59e8a5adad62d2245cbb7b2a8685b1a9a717
	VersionWithBuildMetadata bool
}

// New returns a new Builder
func New() *Builder {
	return &Builder{
		ID: getulid(),
	}
}

// Logs returns the path to the build logs.
//
// Set after Up is called (otherwise "").
func (b *Builder) Logs(appName string) string {
	return filepath.Join(b.LogsDir, appName, b.ID)
}

// Context contains information about the application
type Context struct {
	Manifest   *manifest.Manifest
	AppDir     string
	Components []Component
}

// Component contains the information of a built component
type Component interface {
	Name() string
	Type() string
	URI() string
	Digest() string

	PrepareBuild(*Context) error
	Build(context.Context, *AppContext) error
}

// AppContext contains state information carried across various duffle stage boundaries
type AppContext struct {
	Bldr *Builder
	Ctx  *Context
	Log  io.WriteCloser
	ID   string
}

// PrepareBuild prepares a build
func (b *Builder) PrepareBuild(bldr *Builder, mfst *manifest.Manifest, appDir string, components []Component) (*AppContext, *bundle.Bundle, error) {
	ctx := &Context{
		AppDir:     appDir,
		Components: components,
		Manifest:   mfst,
	}
	bf := &bundle.Bundle{
		Name:        ctx.Manifest.Name,
		Description: ctx.Manifest.Description,
		Images:      []bundle.Image{},
		Keywords:    ctx.Manifest.Keywords,
		Maintainers: ctx.Manifest.Maintainers,
		Parameters:  ctx.Manifest.Parameters,
		Credentials: ctx.Manifest.Credentials,
	}

	for _, c := range ctx.Components {
		if err := c.PrepareBuild(ctx); err != nil {
			return nil, nil, err
		}

		if c.Name() == "cnab" {
			ii := bundle.InvocationImage{}
			ii.Image = c.URI()
			ii.ImageType = c.Type()
			bf.InvocationImages = []bundle.InvocationImage{ii}
			//bf.Version = strings.Split(c.URI(), ":")[1]
			baseVersion := mfst.Version
			if baseVersion == "" {
				baseVersion = "0.1.0"
			}
			newver, err := b.version(baseVersion, strings.Split(c.URI(), ":")[1])
			if err != nil {
				return nil, nil, err
			}
			bf.Version = newver
		} else {
			bundleImage := bundle.Image{Description: c.Name()}
			bundleImage.Image = c.URI()
			bf.Images = append(bf.Images, bundleImage)
		}
	}

	app := &AppContext{
		ID:   bldr.ID,
		Bldr: bldr,
		Ctx:  ctx,
		Log:  os.Stdout,
	}

	return app, bf, nil
}

func (b *Builder) version(baseVersion, sha string) (string, error) {
	sv, err := semver.NewVersion(baseVersion)
	if err != nil {
		return baseVersion, err
	}

	if b.VersionWithBuildMetadata {
		newsv, err := sv.SetMetadata(sha)
		if err != nil {
			return baseVersion, err
		}
		return newsv.String(), nil
	}

	return sv.String(), nil
}

// Build passes the context of each component to its respective builder
func (b *Builder) Build(ctx context.Context, app *AppContext) error {
	if err := buildComponents(ctx, app); err != nil {
		return fmt.Errorf("error building components: %v", err)
	}
	return nil
}

func buildComponents(ctx context.Context, app *AppContext) (err error) {
	errc := make(chan error)

	go func() {
		defer close(errc)
		var wg sync.WaitGroup
		wg.Add(len(app.Ctx.Components))

		for _, c := range app.Ctx.Components {
			go func(c Component) {
				defer wg.Done()
				err = c.Build(ctx, app)
				if err != nil {
					errc <- fmt.Errorf("error building component %v: %v", c.Name(), err)
				}
			}(c)
		}

		wg.Wait()
	}()

	for errc != nil {
		select {
		case err, ok := <-errc:
			if !ok {
				errc = nil
				continue
			}
			return err
		default:
			time.Sleep(time.Second)
		}
	}
	return nil
}
