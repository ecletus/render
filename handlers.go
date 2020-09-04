package render

import (
	"context"
	"io"

	"github.com/moisespsena/template/html/template"

	"github.com/ecletus/core"
)

type key uint8

const (
	contextFormHandlersKey key = iota
	contextScriptHandlersKey
	contextStyleHandlersKey
)

func AddFormHandler(c context.Context, handlers ...*FormHandler) context.Context {
	v := c.Value(contextFormHandlersKey)
	if v == nil {
		c = context.WithValue(c, contextFormHandlersKey, handlers)
	} else {
		c = context.WithValue(c, contextFormHandlersKey, v.(FormHandlers).AppendCopy(handlers...))
	}
	return c
}

func GetFormHandlers(c context.Context) (r FormHandlers) {
	if v := c.Value(contextFormHandlersKey); v != nil {
		r = v.(FormHandlers)
	}
	return
}

type FormHandler struct {
	Name    string
	Handler func(state *FormState, ctx *core.Context) (err error)
}

type FormHandlers []*FormHandler

func (this *FormHandlers) Append(handlers ...*FormHandler) FormHandlers {
	*this = append(*this, handlers...)
	return *this
}

func (this FormHandlers) AppendCopy(handlers ...*FormHandler) FormHandlers {
	return append(this, handlers...)
}

func AddScriptHandler(c context.Context, handlers ...*ScriptHandler) context.Context {
	v := c.Value(contextScriptHandlersKey)
	if v == nil {
		c = context.WithValue(c, contextFormHandlersKey, handlers)
	} else {
		c = context.WithValue(c, contextFormHandlersKey, v.(ScriptHandlers).AppendCopy(handlers...))
	}
	return c
}

func GetScriptHandlers(c context.Context) (r ScriptHandlers) {
	if v := c.Value(contextScriptHandlersKey); v != nil {
		r = v.(ScriptHandlers)
	}
	return
}

type ScriptHandler struct {
	Name    string
	Handler func(state *template.State, ctx *core.Context, w io.Writer) (err error)
}

type ScriptHandlers []*ScriptHandler

func (this *ScriptHandlers) Append(handlers ...*ScriptHandler) ScriptHandlers {
	*this = append(*this, handlers...)
	return *this
}

func (this ScriptHandlers) AppendCopy(handlers ...*ScriptHandler) ScriptHandlers {
	return append(this, handlers...)
}

func AddStyleHandler(c context.Context, handlers ...*StyleHandler) context.Context {
	v := c.Value(contextStyleHandlersKey)
	if v == nil {
		c = context.WithValue(c, contextStyleHandlersKey, handlers)
	} else {
		c = context.WithValue(c, contextStyleHandlersKey, v.(StyleHandlers).AppendCopy(handlers...))
	}
	return c
}

func GetStyleHandlers(c context.Context) (r StyleHandlers) {
	if v := c.Value(contextStyleHandlersKey); v != nil {
		r = v.(StyleHandlers)
	}
	return
}

type StyleHandler struct {
	Name    string
	Handler func(state *template.State, ctx *core.Context, w io.Writer) (err error)
}

type StyleHandlers []*StyleHandler

func (this *StyleHandlers) Append(handlers ...*StyleHandler) StyleHandlers {
	*this = append(*this, handlers...)
	return *this
}

func (this StyleHandlers) AppendCopy(handlers ...*StyleHandler) StyleHandlers {
	return append(this, handlers...)
}
