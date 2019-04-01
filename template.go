package render

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/moisespsena/go-assetfs"
	"github.com/moisespsena-go/default-logger"
	"github.com/moisespsena-go/path-helpers"
	"github.com/moisespsena/template/cache"
	"github.com/moisespsena/template/funcs"
	"github.com/moisespsena/template/html/template"
	"github.com/ecletus/core"
)

var log = defaultlogger.NewLogger(path_helpers.GetCalledDir())

// Template template struct
type Template struct {
	render             *Render
	layout             string
	usingDefaultLayout bool
	funcMaps           []template.FuncMap
	funcValues         []*template.FuncValues
	DebugFiles         bool
}

// Layout set layout for template.
func (tmpl *Template) Layout(name string) *Template {
	tmpl.layout = name
	return tmpl
}

// FuncMap get func maps from tmpl
func (tmpl *Template) funcMapMaker(values *template.FuncValues, context *core.Context) error {
	values.SetDefault("locale", func() string {
		return context.GetLocale()
	})
	values.SetDefault("prefix", func() string {
		return ""
	})
	values.SetDefault("local_url", func(ctx *funcs.Context) func(...string) string {
		prefix := ctx.Get("prefix").String()
		return func(s ...string) string {
			if prefix == "" {
				return context.GenURL(s...)
			}
			return context.GenURL(append([]string{prefix}, s...)...)
		}
	})
	values.SetDefault("local_static_url", func(ctx *funcs.Context) func(...string) string {
		prefix := ctx.Get("prefix").String()
		return func(s ...string) string {
			if prefix == "" {
				return context.GenStaticURL(s...)
			}
			return context.GenStaticURL(append([]string{prefix}, s...)...)
		}
	})
	values.SetDefault("static_url", context.GenStaticURL)
	values.SetDefault("url", context.GenURL)

	values.AppendValues(tmpl.render.funcs)

	if tmpl.render.Config.FuncMapMaker != nil {
		err := tmpl.render.Config.FuncMapMaker(values, tmpl.render, context)
		if err != nil {
			return err
		}
	}

	for _, fm := range tmpl.render.funcMapMakers {
		err := fm(values, tmpl.render, context)
		if err != nil {
			return err
		}
	}

	return values.Append(tmpl.funcMaps...)
}

// Funcs register Funcs for tmpl
func (tmpl *Template) Funcs(funcMaps ...template.FuncMap) *Template {
	tmpl.funcMaps = append(tmpl.funcMaps, funcMaps...)
	return tmpl
}

// Funcs register Funcs for tmpl
func (tmpl *Template) FuncValues(funcValues ...*template.FuncValues) *Template {
	tmpl.funcValues = append(tmpl.funcValues, funcValues...)
	return tmpl
}

// Render render tmpl
func (tmpl *Template) Render(templateName string, obj interface{}, context *core.Context) (template.HTML, error) {
	var funcValues = &template.FuncValues{}

	render := func(name string, require bool, objs ...interface{}) (template.HTML, error) {
		var (
			err       error
			renderObj interface{}
		)
		if len(objs) == 0 {
			// default obj
			renderObj = obj
		} else {
			// overwrite obj
			renderObj, objs = objs[0], objs[1:]
		}

		var exectr *template.Executor
		if exectr, err = tmpl.GetExecutor(name); err == nil {
			result := bytes.NewBufferString("")
			exectr = exectr.FuncsValues(funcValues)
			if len(objs) > 0 {
				for i, max := 0, len(objs); i < max; i++ {
					switch ot := objs[i].(type) {
					case template.LocalData:
						if i == 0 {
							exectr.Local = &ot
						} else {
							exectr.Local.Merge(ot)
						}
					case map[interface{}]interface{}:
						exectr.Local.Merge(ot)
					default:
						exectr.Local.Set(objs[i], objs[i+1])
						i++
					}
				}
			}
			if err = exectr.Execute(result, renderObj); err == nil {
				return template.HTML(result.String()), err
			}
		}

		if err != nil {
			if et, ok := err.(*template.ErrorWithTrace); ok {
				log.Error(err.Error() + "\n" + string(et.Trace()))
			} else {
				log.Error(err)
			}
		}

		return "", err
	}

	require := func(name string, objs ...interface{}) (template.HTML, error) {
		return render(name, true, objs...)
	}

	include := func(name string, objs ...interface{}) (template.HTML, error) {
		return render(name, false, objs...)
	}

	tmpl.funcMapMaker(funcValues, context)

	// funcMaps
	funcValues.Set("render", require)
	funcValues.Set("require", require)
	funcValues.Set("include", include)
	funcValues.Set("yield", func() (template.HTML, error) {
		return require(templateName)
	})

	layout := tmpl.layout
	usingDefaultLayout := false

	if layout == "" && tmpl.usingDefaultLayout {
		usingDefaultLayout = true
		layout = tmpl.render.DefaultLayout
	}

	if layout != "" {
		name := filepath.Join("layouts", layout)
		data, err := require(name)
		if err == nil {
			return data, nil
		} else if !usingDefaultLayout {
			err = fmt.Errorf("Failed to render layout: '%v.tmpl', got error: %v", filepath.Join("layouts", tmpl.layout), err)
			fmt.Println(err)
			return template.HTML(""), err
		}
	}

	return require(templateName)
}

// Execute execute tmpl
func (tmpl *Template) Execute(templateName string, obj interface{}, context *core.Context) error {
	result, err := tmpl.Render(templateName, obj, context)
	if err == nil {
		w := context.Writer
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "text/html")
		}

		_, err = w.Write([]byte(result))
	}
	return err
}

func (tmpl *Template) findTemplate(name string) (assetfs.AssetInterface, error) {
	return tmpl.render.Asset(name + ".tmpl")
}

func (tmpl *Template) GetExecutor(name string) (*template.Executor, error) {
	return cache.Cache.LoadOrStore(name, func(name string) (*template.Executor, error) {
		asset, err := tmpl.findTemplate(name)
		if err != nil {
			return nil, fmt.Errorf("failed to find template: %q", name)
		}
		t, err := template.New(name).SetPath(asset.GetPath()).Parse(asset.GetString())
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %q: %v", name, err)
		}

		if tmpl.DebugFiles {
			log.Debug(fmt.Sprintf("{%v} %v", name, asset.GetPath()))
		}

		return t.CreateExecutor(), nil
	})
}
