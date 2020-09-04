// Package render support to render templates by your control.
package render

import (
	"bytes"
	"os"
	"strings"

	path_helpers "github.com/moisespsena-go/path-helpers"

	"github.com/ecletus/about"
	"github.com/moisespsena-go/assetfs"

	"github.com/ecletus/core"
	"github.com/ecletus/session"
	"github.com/microcosm-cc/bluemonday"
	"github.com/moisespsena-go/bid"
	"github.com/moisespsena/template/html/template"
)

var pkg = path_helpers.GetCalledDir()

// DefaultLayout default layout name
const DefaultLayout = "application"

func DefaultLocale() string {
	locale := os.Getenv("LANG")
	if locale != "" {
		locale = strings.Split(locale, ".")[0]
	}
	return locale
}

type FuncMapMaker func(values *template.FuncValues, render *Render, context *core.Context) error

type FormState struct {
	Name string
	Body string
}

// Config render config
type Config struct {
	PageHandlers
	DefaultLayout string
	FuncMapMaker  FuncMapMaker
	AssetFS       assetfs.Interface
	DebugFiles    bool
	DefaultLocale string
	Abouter       about.SiteAbouter
}

func (this *PageHandlers) GetFormHandlers() *FormHandlers {
	return &this.FormHandlers
}

func (this *PageHandlers) GetScriptHandlers() *ScriptHandlers {
	return &this.ScriptHandlers
}

func (this *PageHandlers) GetStyleHandlers() *StyleHandlers {
	return &this.StyleHandlers
}

type PageHandler interface {
	GetFormHandlers() *FormHandlers
	GetScriptHandlers() *ScriptHandlers
	GetStyleHandlers() *StyleHandlers
}

type PageHandlers struct {
	FormHandlers   FormHandlers
	ScriptHandlers ScriptHandlers
	StyleHandlers  StyleHandlers
}

type funcMapMakers struct {
	m     map[string]FuncMapMaker
	names []string
}

// Render the render struct.
type Render struct {
	*Config
	funcMapMakers funcMapMakers
	funcs         template.FuncValues
}

func Context(s *template.State, defaul ...*core.Context) *core.Context {
	switch ctx := s.Context().(type) {
	case *core.Context:
		return ctx
	case core.ContextGetter:
		return ctx.GetContext()
	}
	for _, d := range defaul {
		return d
	}
	return nil
}

// New initalize the render struct.
func New(config *Config) *Render {
	if config == nil {
		config = &Config{}
	}

	if config.DefaultLocale == "" {
		config.DefaultLocale = DefaultLocale()
	}

	render := &Render{Config: config}

	render.RegisterFuncMapMaker("qor_context", func(funcs *template.FuncValues, render *Render, ctx *core.Context) error {
		funcs.SetDefault("qor_context", func(s *template.State) *core.Context {
			return Context(s)
		})

		funcs.SetDefault("current_locale", func(s *template.State) string {
			if cookie, err := Context(s).Request.Cookie("locale"); err == nil {
				return cookie.Value
			}
			return config.DefaultLocale
		})

		funcs.SetDefault("flashes", func(s *template.State) []session.Message {
			return Context(s).SessionManager().Flashes()
		})

		funcs.SetDefault("about", func(s *template.State) about.Abouter {
			return config.Abouter.About(Context(s).Site)
		})

		i18nCtx := ctx.GetI18nContext()

		funcs.Set("t", func(key string, defaul ...interface{}) template.HTML {
			return template.HTML(i18nCtx.T(key).DefaultArgs(defaul...).Get())
		})

		funcs.Set("tt", func(key string, data interface{}, defaul ...interface{}) template.HTML {
			return template.HTML(i18nCtx.TT(key).DefaultArgs(defaul...).Data(data).Get())
		})

		funcs.SetDefault("errors", func(s *template.State) []string {
			return Context(s).GetErrorsTS()
		})

		funcs.SetDefault("render_scripts", func(s *template.State) (r template.HTML) {
			var (
				c = Context(s)
				w bytes.Buffer
			)

			for _, h := range []ScriptHandlers{render.ScriptHandlers, GetScriptHandlers(s.Context())} {
				for _, h := range h {
					if err := h.Handler(s, c, &w); err != nil {
						w.WriteString("[[render execute script handler `" + h.Name + "` failed: " + err.Error() + "]]")
						break
					}
				}
			}
			return template.HTML(w.String())
		})

		funcs.SetDefault("render_styles", func(s *template.State) (r template.HTML) {
			var (
				c = Context(s)
				w bytes.Buffer
			)

			for _, h := range []StyleHandlers{render.StyleHandlers, GetStyleHandlers(s.Context())} {
				for _, h := range h {
					if err := h.Handler(s, c, &w); err != nil {
						w.WriteString("[[render execute style handler `" + h.Name + "` failed: " + err.Error() + "]]")
						break
					}
				}
			}
			return template.HTML(w.String())
		})

		funcs.SetDefault("form", func(s *template.State, name string, pipes ...interface{}) template.HTML {
			var c = Context(s)

			state := &FormState{name, s.Exec(name, pipes...)}
			for _, h := range render.FormHandlers.AppendCopy(GetFormHandlers(s.Context())...) {
				if err := h.Handler(state, c); err != nil {
					return template.HTML("[[render execute form handler `" + h.Name + "` for `" + name + "` form failed: " + err.Error() + "]]")
				}
			}
			return template.HTML(state.Body)
		})

		funcs.SetDefault("must_config_get", func(configor core.Configor, key interface{}) (v interface{}) {
			v, _ = configor.ConfigGet(key)
			return v
		})

		funcs.SetDefault("media_url", func(pth string, storageName ...string) string {
			var sname = "default"
			for _, sname = range storageName {
			}
			return ctx.MediaURL(sname, pth)
		})
		return nil
	})

	htmlSanitizer := bluemonday.UGCPolicy()
	render.RegisterFuncMap("raw", func(str string) template.HTML {
		return template.HTML(htmlSanitizer.Sanitize(str))
	})
	render.RegisterFuncMap("genid", func() string {
		return bid.New().String()
	})

	return render
}

// SetAssetFS set asset fs for render
func (this *Render) SetAssetFS(assetFS assetfs.Interface) {
	this.AssetFS = assetFS
}

// Layout set layout for template.
func (this *Render) Layout(name string) (t *Template) {
	t = this.Template()
	t.Layout = name
	return
}

// Funcs set helper functions for template with default "application" layout.
func (this *Render) Funcs() template.FuncValues {
	return this.funcs
}

// FuncsPtr set helper functions for template with default "application" layout.
func (this *Render) FuncsPtr() *template.FuncValues {
	return &this.funcs
}

// Execute render template with default "application" layout.
func (this *Render) Execute(name string, data interface{}, context *core.Context) error {
	return this.Template().Execute(name, data, context)
}

func (this *Render) Template() *Template {
	t := NewTemplate(this)
	t.UsingDefaultLayout = true
	t.DebugFiles = this.Config.DebugFiles
	t.DefaultLayout = this.DefaultLayout
	return t
}

// RegisterFuncMap register FuncMap for render.
func (this *Render) RegisterFuncMap(name string, fc interface{}) {
	err := this.funcs.Set(name, fc)
	if err != nil {
		panic(err)
	}
}

// RegisterFuncMapMaker register FuncMap for render.
func (this *Render) RegisterFuncMapMaker(name string, fm FuncMapMaker) {
	if this.funcMapMakers.m == nil {
		this.funcMapMakers.m = make(map[string]FuncMapMaker)
	}
	if _, ok := this.funcMapMakers.m[name]; !ok {
		this.funcMapMakers.names = append(this.funcMapMakers.names, name)
	}
	this.funcMapMakers.m[name] = fm
}

// Asset get content from AssetFS by name
func (this *Render) Asset(name string) (asset assetfs.AssetInterface, err error) {
	return this.AssetFS.Asset(name)
}
