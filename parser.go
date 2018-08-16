package render

import "github.com/moisespsena/template/html/template"

func Parse(name string, content string) (*template.Template, error) {
	return template.New(name).Parse(content)
}
