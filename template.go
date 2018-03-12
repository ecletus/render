package render

import (
	"bytes"
	"fmt"
	"path/filepath"
	"github.com/qor/qor"
	"github.com/moisespsena/template/cache"
	"github.com/moisespsena/template/funcs"
	"github.com/moisespsena/template/html/template"
)

// Template template struct
type Template struct {
	render             *Render
	layout             string
	usingDefaultLayout bool
	funcMaps           []template.FuncMap
	funcValues         []*template.FuncValues
}

// Layout set layout for template.
func (tmpl *Template) Layout(name string) *Template {
	tmpl.layout = name
	return tmpl
}

// FuncMap get func maps from tmpl
func (tmpl *Template) funcMapMaker(values *template.FuncValues, context *qor.Context) error {
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
func (tmpl *Template) Funcs(funcMaps ... template.FuncMap) *Template {
	tmpl.funcMaps = append(tmpl.funcMaps, funcMaps...)
	return tmpl
}

// Funcs register Funcs for tmpl
func (tmpl *Template) FuncValues(funcValues ... *template.FuncValues) *Template {
	tmpl.funcValues = append(tmpl.funcValues, funcValues...)
	return tmpl
}

// Render render tmpl
func (tmpl *Template) Render(templateName string, obj interface{}, context *qor.Context) (template.HTML, error) {
	var (
		content    []byte
		t          *template.Template
		err        error
		funcValues = &template.FuncValues{}
		render     = func(name string, objs ...interface{}) (template.HTML, error) {
			var (
				err           error
				renderObj     interface{}
				renderContent []byte
			)

			if len(objs) == 0 {
				// default obj
				renderObj = obj
			} else {
				// overwrite obj
				for _, o := range objs {
					renderObj = o
					break
				}
			}

			if renderContent, err = tmpl.findTemplate(name); err == nil {
				var partialTemplate *template.Template
				result := bytes.NewBufferString("")
				if partialTemplate, err = template.New(filepath.Base(name)).Parse(string(renderContent)); err == nil {
					if err = partialTemplate.CreateExecutor().FuncsValues(funcValues).Execute(result, renderObj); err == nil {
						return template.HTML(result.String()), err
					}
				}
			} else {
				err = fmt.Errorf("failed to find template: %v", name)
			}

			if err != nil {
				fmt.Println(err)
			}

			return "", err
		}
	)

	tmpl.funcMapMaker(funcValues, context)

	// funcMaps
	funcValues.Set("render", render)
	funcValues.Set("yield", func() (template.HTML, error) {
		return render(templateName)
	})

	layout := tmpl.layout
	usingDefaultLayout := false

	if layout == "" && tmpl.usingDefaultLayout {
		usingDefaultLayout = true
		layout = tmpl.render.DefaultLayout
	}

	if layout != "" {
		name := filepath.Join("layouts", layout)
		executor, err := tmpl.GetExecutor(name)
		if err == nil {
			var tpl bytes.Buffer
			if err = executor.FuncsValues(funcValues).Execute(&tpl, obj); err == nil {
				return template.HTML(tpl.String()), nil
			}
		} else if !usingDefaultLayout {
			err = fmt.Errorf("Failed to render layout: '%v.tmpl', got error: %v", filepath.Join("layouts", tmpl.layout), err)
			fmt.Println(err)
			return template.HTML(""), err
		}
	}

	if content, err = tmpl.findTemplate(templateName); err == nil {
		if t, err = template.New(templateName).Parse(string(content)); err == nil {
			var tpl bytes.Buffer
			if err = t.CreateExecutor().FuncsValues(funcValues).Execute(&tpl, obj); err == nil {
				return template.HTML(tpl.String()), nil
			}
		}
	} else {
		err = fmt.Errorf("failed to find template: %v", templateName)
	}

	if err != nil {
		fmt.Println(err)
	}
	return template.HTML(""), err
}

// Execute execute tmpl
func (tmpl *Template) Execute(templateName string, obj interface{}, context *qor.Context) error {
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

func (tmpl *Template) findTemplate(name string) ([]byte, error) {
	return tmpl.render.Asset(name + ".tmpl")
}

func (tmpl *Template) GetExecutor(name string) (*template.Executor, error) {
	return cache.Cache.LoadOrStore(name, func(name string) (*template.Executor, error) {
		data, err := tmpl.findTemplate(name)
		if err != nil {
			return nil, fmt.Errorf("failed to find template: %q", name)
		}
		t, err := template.New(name).Parse(string(data))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %q: %v", name, err)
		}
		return t.CreateExecutor(), nil
	})
}
