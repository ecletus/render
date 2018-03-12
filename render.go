// Package render support to render templates by your control.
package render

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/qor/qor"
	"github.com/qor/assetfs"
	"github.com/qor/qor/utils"
	"github.com/moisespsena/template/html/template"
)

// DefaultLayout default layout name
const DefaultLayout = "application"

// DefaultViewPath default view path
const DefaultViewPath = "app/views"

type FuncMapMaker func(values *template.FuncValues, render *Render, context *qor.Context) error

// Config render config
type Config struct {
	ViewPaths       []string
	DefaultLayout   string
	FuncMapMaker    FuncMapMaker
	AssetFileSystem assetfs.Interface
}


// Render the render struct.
type Render struct {
	*Config
	funcMapMakers map[string]FuncMapMaker
	funcs *template.FuncValues
}

// New initalize the render struct.
func New(config *Config, viewPaths ...string) *Render {
	if config == nil {
		config = &Config{}
	}

	if config.DefaultLayout == "" {
		config.DefaultLayout = DefaultLayout
	}

	if config.AssetFileSystem == nil {
		config.AssetFileSystem = assetfs.AssetFS().NameSpace("views")
	}

	config.ViewPaths = append(append(config.ViewPaths, viewPaths...), DefaultViewPath)

	render := &Render{funcs: &template.FuncValues{}, Config: config}

	for _, viewPath := range config.ViewPaths {
		render.RegisterViewPath(viewPath)
	}

	return render
}

// RegisterViewPath register view path
func (render *Render) RegisterViewPath(paths ...string) {
	for _, pth := range paths {
		if filepath.IsAbs(pth) {
			render.ViewPaths = append(render.ViewPaths, pth)
			render.AssetFileSystem.RegisterPath(pth)
		} else {
			if absPath, err := filepath.Abs(pth); err == nil && isExistingDir(absPath) {
				render.ViewPaths = append(render.ViewPaths, absPath)
				render.AssetFileSystem.RegisterPath(absPath)
			} else if isExistingDir(filepath.Join(utils.AppRoot, "vendor", pth)) {
				render.AssetFileSystem.RegisterPath(filepath.Join(utils.AppRoot, "vendor", pth))
			} else {
				for _, gopath := range strings.Split(os.Getenv("GOPATH"), ":") {
					if p := filepath.Join(gopath, "src", pth); isExistingDir(p) {
						render.ViewPaths = append(render.ViewPaths, p)
						render.AssetFileSystem.RegisterPath(p)
					}
				}
			}
		}
	}
}

// PrependViewPath prepend view path
func (render *Render) PrependViewPath(paths ...string) {
	for _, pth := range paths {
		if filepath.IsAbs(pth) {
			render.ViewPaths = append([]string{pth}, render.ViewPaths...)
			render.AssetFileSystem.PrependPath(pth)
		} else {
			if absPath, err := filepath.Abs(pth); err == nil && isExistingDir(absPath) {
				render.ViewPaths = append([]string{absPath}, render.ViewPaths...)
				render.AssetFileSystem.PrependPath(absPath)
			} else if isExistingDir(filepath.Join(utils.AppRoot, "vendor", pth)) {
				render.AssetFileSystem.PrependPath(filepath.Join(utils.AppRoot, "vendor", pth))
			} else {
				for _, gopath := range strings.Split(os.Getenv("GOPATH"), ":") {
					if p := filepath.Join(gopath, "src", pth); isExistingDir(p) {
						render.ViewPaths = append([]string{p}, render.ViewPaths...)
						render.AssetFileSystem.PrependPath(p)
					}
				}
			}
		}
	}
}

// SetAssetFS set asset fs for render
func (render *Render) SetAssetFS(assetFS assetfs.Interface) {
	for _, viewPath := range render.ViewPaths {
		assetFS.RegisterPath(viewPath)
	}

	render.AssetFileSystem = assetFS
}

// Layout set layout for template.
func (render *Render) Layout(name string) *Template {
	return &Template{render: render, layout: name}
}

// Funcs set helper functions for template with default "application" layout.
func (render *Render) Funcs() *template.FuncValues {
	return render.funcs
}

// Execute render template with default "application" layout.
func (render *Render) Execute(name string, data interface{}, context *qor.Context) error {
	tmpl := &Template{render: render, usingDefaultLayout: true}
	return tmpl.Execute(name, data, context)
}

func (render *Render) Template() *Template {
	return &Template{render: render, usingDefaultLayout: true}
}

// RegisterFuncMap register FuncMap for render.
func (render *Render) RegisterFuncMap(name string, fc interface{}) {
	err := render.funcs.Set(name, fc)
	if err != nil {
		panic(err)
	}
}

// RegisterFuncMapMaker register FuncMap for render.
func (render *Render) RegisterFuncMapMaker(name string, fm FuncMapMaker) {
	if render.funcMapMakers == nil {
		render.funcMapMakers = make(map[string]FuncMapMaker)
	}
	render.funcMapMakers[name] = fm
}

// Asset get content from AssetFS by name
func (render *Render) Asset(name string) ([]byte, error) {
	return render.AssetFileSystem.Asset(name)
}
