package flotilla

import "reflect"

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
		url, err := ctx.urlfor(route, external, params)
		if err != nil {
			return err
		}
		return url
	}
	return newError("Unable to return a url.")
}

func (t TData) ContextProcessor(fn reflect.Value, ctx *Ctx) (string, error) {
	res, err := call(fn, ctx)
	if err != nil {
		return "", err
	}
	if ret, ok := res.(string); ok {
		return ret, nil
	}
	return "", newError("No string returned.")
}

func (t TData) ContextProcessors(ctx *Ctx) {
	if ctx.ctxprcss != nil {
		for k, v := range ctx.ctxprcss {
			ret, err := t.ContextProcessor(v, ctx)
			if err != nil {
				t[k] = err
			}
			t[k] = ret
		}
	}
}
