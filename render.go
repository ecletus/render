// Package render support to render templates by your control.
package render

import (
	"os"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/moisespsena/go-assetfs"
	"github.com/moisespsena/template/html/template"
	"github.com/aghape/aghape"
	"github.com/aghape/session"
	"gopkg.in/mgo.v2/bson"
)

// DefaultLayout default layout name
const DefaultLayout = "application"

func DefaultLocale() string {
	locale := os.Getenv("LANG")
	if locale != "" {
		locale = strings.Split(locale, ".")[0]
	}
	return locale
}

var DEFAULT_LOCALE = DefaultLocale()

type FuncMapMaker func(values *template.FuncValues, render *Render, context *qor.Context) error

// Config render config
type Config struct {
	DefaultLayout string
	FuncMapMaker  FuncMapMaker
	AssetFS       assetfs.Interface
	DebugFiles    bool
}

// Render the render struct.
type Render struct {
	*Config
	funcMapMakers map[string]FuncMapMaker
	funcs         *template.FuncValues
}

// New initalize the render struct.
func New(config *Config) *Render {
	if config == nil {
		config = &Config{}
	}

	render := &Render{funcs: &template.FuncValues{}, Config: config}

	render.RegisterFuncMapMaker("qor_context", func(funcs *template.FuncValues, render *Render, context *qor.Context) error {
		funcs.SetDefault("qor_context", func() *qor.Context {
			return context
		})

		funcs.SetDefault("current_locale", func() string {
			if cookie, err := context.Request.Cookie("locale"); err == nil {
				return cookie.Value
			}
			return DEFAULT_LOCALE
		})

		funcs.SetDefault("flashes", func() []session.Message {
			return context.SessionManager().Flashes()
		})

		ctx := context.GetI18nContext()
		funcs.SetDefault("t", func(key string, args ...interface{}) template.HTML {
			return template.HTML(ctx.T(key).DefaultAndDataFromArgs(args...).Get())
		})

		return nil
	})

	htmlSanitizer := bluemonday.UGCPolicy()
	render.RegisterFuncMap("raw", func(str string) template.HTML {
		return template.HTML(htmlSanitizer.Sanitize(str))
	})
	render.RegisterFuncMap("genid", func() string {
		return bson.NewObjectId().Hex()
	})

	return render
}

// SetAssetFS set asset fs for render
func (render *Render) SetAssetFS(assetFS assetfs.Interface) {
	render.AssetFS = assetFS
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
	tmpl := &Template{render: render, usingDefaultLayout: true, DebugFiles: render.Config.DebugFiles}
	return tmpl.Execute(name, data, context)
}

func (render *Render) Template() *Template {
	return &Template{render: render, usingDefaultLayout: true, DebugFiles: render.Config.DebugFiles}
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
func (render *Render) Asset(name string) (assetfs.AssetInterface, error) {
	return render.AssetFS.Asset(name)
}
