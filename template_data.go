package flotilla

import (
	"fmt"
	"html/template"
	"reflect"
)

type (
	// TData is a map sent to and accessible within the template, by the
	// builtin rendertemplate function.
	TData map[string]interface{}

	contextprocessor struct {
		name   string
		format string
		fn     reflect.Value
	}
)

func templatedata(any interface{}) TData {
	if rcvd, ok := any.(map[string]interface{}); ok {
		td := rcvd
		return td
	} else {
		td := make(map[string]interface{})
		td["Any"] = any
		return td
	}
}

func TemplateData(ctx *Ctx, any interface{}) TData {
	td := templatedata(any)
	td["Ctx"] = ctx
	td["Request"] = ctx.Request
	td["Session"] = ctx.Session
	for k, v := range ctx.Data {
		td[k] = v
	}
	td["Flash"] = allflashmessages(ctx)
	td.ContextProcessors(ctx)
	return td
}

// GetFlashMessages gets flash messages stored with TData by category.
func (t TData) GetFlashMessages(categories ...string) []string {
	var ret []string
	if fls, ok := t["Flash"].(map[string]string); ok {
		for k, v := range fls {
			if existsIn(k, categories) {
				ret = append(ret, v)
			}
		}
	}
	return ret
}

func (t TData) UrlFor(route string, external bool, params ...string) string {
	if ctx, ok := t["Ctx"].(*Ctx); ok {
		ret, err := ctx.Call("urlfor", route, external, params)
		if err != nil {
			return newError(fmt.Sprint("%s", err)).Error()
		}
		return ret.(string)
	}
	return newError("Unable to return a url.").Error()
}

func ContextProcessor(name string, format string, function interface{}) *contextprocessor {
	return &contextprocessor{name: name,
		format: format,
		fn:     valueFunc(function),
	}
}

func (t TData) HtmlContextProcessor(cp *contextprocessor, ctx *Ctx) template.HTML {
	res, err := call(cp.fn, ctx)
	if err != nil {
		return template.HTML(fmt.Sprintf("%s", err))
	}
	if ret, ok := res.(template.HTML); ok {
		return ret
	}
	return template.HTML("No HTML returned.")
}

func (t TData) StringContextProcessor(cp *contextprocessor, ctx *Ctx) string {
	res, err := call(cp.fn, ctx)
	if err != nil {
		return fmt.Sprintf("%s", err)
	}
	if ret, ok := res.(string); ok {
		return ret
	}
	return "No string returned."
}

func (t TData) ContextProcessors(ctx *Ctx) {
	for _, cp := range ctx.ctxprcss {
		switch cp.format {
		case "html":
			t[cp.name] = t.HtmlContextProcessor(cp, ctx)
		case "string":
			t[cp.name] = t.StringContextProcessor(cp, ctx)
		default:
			t[cp.name] = t.StringContextProcessor(cp, ctx)
		}
	}
}
