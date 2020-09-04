package render

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"io"

	"github.com/moisespsena/template/render"

	"github.com/ecletus/core"
	"github.com/moisespsena-go/assetfs"
	defaultlogger "github.com/moisespsena-go/default-logger"
	oscommon "github.com/moisespsena-go/os-common"
	path_helpers "github.com/moisespsena-go/path-helpers"
	"github.com/moisespsena/template/cache"
	"github.com/moisespsena/template/funcs"
	"github.com/moisespsena/template/html/template"
)

var log = defaultlogger.GetOrCreateLogger(path_helpers.GetCalledDir())

// Template template struct
type Template struct {
	render.Template
	render     *Render
	DebugFiles bool
}

func NewTemplate(render *Render) (t *Template) {
	t = new(Template)
	t.render = render
	t.Template.GetExecutor = t.getExecutor
	return
}

// FuncMap get func maps from tmpl
func (this *Template) prepare(values *template.FuncValues, context *core.Context) error {
	values.Start().AppendValues(this.render.funcs)

	if this.render.Config.FuncMapMaker != nil {
		err := this.render.Config.FuncMapMaker(values, this.render, context)
		if err != nil {
			return err
		}
	}

	for _, name := range this.render.funcMapMakers.names {
		fm := this.render.funcMapMakers.m[name]
		err := fm(values, this.render, context)
		if err != nil {
			return err
		}
	}

	values.Set("locale", func() string {
		return context.GetLocale()
	})
	values.Set("prefix", func() string {
		return ""
	})
	values.Set("local_url", func(ctx *funcs.Context) func(...string) string {
		prefix := ctx.Get("prefix").String()
		return func(s ...string) string {
			if prefix == "" {
				return context.Path(s...)
			}
			return context.Path(append([]string{prefix}, s...)...)
		}
	})
	values.Set("local_static_url", func(ctx *funcs.Context) func(...string) string {
		prefix := ctx.Get("prefix").String()
		return func(s ...string) string {
			if prefix == "" {
				return context.JoinStaticURL(s...)
			}
			return context.JoinStaticURL(append([]string{prefix}, s...)...)
		}
	})
	values.Set("static_url", context.JoinStaticURL)
	values.Set("url", context.Path)

	return values.Append(this.Funcs...)
}

// Render render tmpl
func (this Template) RenderW(state *template.State, w io.Writer, templateName string, obj interface{}, ctx *core.Context, lang ...string) (err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, pkg+": RenderW `"+templateName+"` failed")
		}
	}()
	var values template.FuncValues
	if err = (&this).prepare(&values, ctx); err != nil {
		return
	}
	var Context context.Context = ctx
	this.FuncValues.Append(values)
	return this.Template.Render(state, w, Context, templateName, obj, lang...)
}

// Render render tmpl
func (this Template) Render(state *template.State, templateName string, obj interface{}, ctx *core.Context, lang ...string) (s template.HTML, err error) {
	var w bytes.Buffer
	if err = this.RenderW(state, &w, templateName, obj, ctx, lang...); err != nil {
		return
	}
	return template.HTML(w.String()), nil
}

// Execute execute tmpl
func (this *Template) Execute(templateName string, obj interface{}, context *core.Context) (err error) {
	var w bytes.Buffer
	if err = this.RenderW(nil, &w, templateName, obj, context); err == nil {
		cw := context.Writer
		if cw.Header().Get("Content-Type") == "" {
			cw.Header().Set("Content-Type", "text/html")
		}

		_, err = cw.Write(w.Bytes())
	}
	return
}

func (this *Template) findTemplate(name string) (assetfs.AssetInterface, error) {
	return this.render.Asset(name + ".tmpl")
}

func (this *Template) getExecutor(name string) (*template.Executor, error) {
	return cache.Cache.LoadOrStore(name, func(name string) (*template.Executor, error) {
		asset, err := this.findTemplate(name)
		if err != nil {
			if pathErr, ok := err.(*oscommon.PathError); ok {
				pathErr.AddMessage(fmt.Sprintf("failed to find template: %q", name))
				return nil, err
			}
			return nil, fmt.Errorf("failed to find template %q: %q", name, err.Error())
		}
		var data string
		if data, err = assetfs.DataS(asset); err != nil {
			return nil, err
		}
		t, err := template.New(name).SetPath(asset.Path()).Parse(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %q: %v", name, err)
		}

		if this.DebugFiles {
			log.Debug(fmt.Sprintf("{%v} %v", name, asset.Path()))
		}

		return t.CreateExecutor(), nil
	})
}

func (this Template) SetLayout(layout string) *Template {
	this.Layout = layout
	return &this
}

func (this Template) SetFuncValues(fv ...template.FuncValues) *Template {
	this.FuncValues.Append(fv...)
	return &this
}

func (this Template) SetFuncs(fv ...template.FuncMap) *Template {
	this.Funcs.Append(fv...)
	return &this
}