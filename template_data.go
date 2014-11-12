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

func (t TData) ContextProcessor(name string, fn reflect.Value, ctx *Ctx) error {
	res, err := call(fn, ctx)
	if err != nil {
		t[name] = fmt.Sprintf("%s", err)
		return nil
	}
	if ret, ok := res.(template.HTML); ok {
		t[name] = ret
		return nil
	}
	if ret, ok := res.(string); ok {
		t[name] = ret
		return nil
	}
	return newError("Context processor could not be run")
}

func (t TData) ContextProcessors(ctx *Ctx) {
	for k, fn := range ctx.ctxprcss {
		err := t.ContextProcessor(k, fn, ctx)
		if err != nil {
			t[k] = err.Error()
		}
	}
}
