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
	ctxcopy := ctx.Copy()
	td := templatedata(any)
	td["Ctx"] = ctxcopy
	td["Request"] = ctx.Request
	td["Session"] = ctx.Session
	for k, v := range ctx.Data {
		td[k] = v
	}
	td["Flash"] = allflashmessages(ctx)
	td.contextprocessors(ctxcopy)
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
		ret, err := ctx.Call("urlfor", ctx, route, external, params)
		if err != nil {
			return newError(fmt.Sprint("%s", err)).Error()
		}
		return ret.(string)
	}
	return fmt.Sprintf("Unable to return a url from: %s, %s, external(%t)", route, params, external)
}

// HTML will call the context processor by name return html, html formatted error,
// or html formatted notice that the processor could not return an html value.
func (t TData) HTML(name string) template.HTML {
	if fn, ok := t.ctxprc(name); ok {
		res, err := call(fn)
		if err != nil {
			return template.HTML(err.Error())
		}
		if ret, ok := res.(template.HTML); ok {
			return ret
		}
	}
	return template.HTML(fmt.Sprintf("<p>context processor %s unprocessable by HTML</p>", name))
}

// STRING will call the context processor by name, returning a string value, an
// error string value, or a string indicating that the processor could not return
// a string value.
func (t TData) STRING(name string) string {
	if fn, ok := t.ctxprc(name); ok {
		res, err := call(fn)
		if err != nil {
			return err.Error()
		}
		if ret, ok := res.(string); ok {
			return ret
		}
	}
	return fmt.Sprintf("context processor %s unprocessable by STRING", name)
}

// CALL will call the context processor by name, returning an interface{} or error.
func (t TData) CALL(name string) interface{} {
	if fn, ok := t.ctxprc(name); ok {
		if res, err := call(fn); err == nil {
			return res
		} else {
			return err
		}
	}
	return fmt.Sprintf("context processor %s cannot be processed by CALL", name)
}

func (t TData) ctxprc(name string) (reflect.Value, bool) {
	if fn, ok := t[name]; ok {
		if fn, ok := fn.(reflect.Value); ok {
			return fn, true
		}
	}
	return reflect.Value{}, false
}

func (t TData) contextprocessor(fn reflect.Value, ctx *Ctx) reflect.Value {
	newfn := func() (interface{}, error) {
		return call(fn, ctx)
	}
	return valueFunc(newfn)
}

func (t TData) contextprocessors(ctxcopy *Ctx) {
	for k, fn := range ctxcopy.processors {
		t[k] = t.contextprocessor(fn, ctxcopy)
	}
}
