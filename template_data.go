package flotilla

import "fmt"

type (
	// TData is a map sent to and accessible within the template, by the
	// builtin rendertemplate function.
	TData map[string]interface{}

	ctxprc func(*Ctx) string
)

func templatedata(any interface{}) TData {
	if rcvd, ok := any.(map[string]interface{}); ok {
		td := rcvd
		fmt.Printf("any IS a map\n")
		return td
	} else {
		td := make(map[string]interface{})
		td["Any"] = any
		fmt.Printf("any IS NOT a map\n")
		return td
	}
}

func TemplateData(ctx *Ctx, any interface{}) TData {
	td := templatedata(any)
	td["Ctx"] = ctx
	td["Request"] = ctx.Request
	td["Session"] = ctx.Session
	td["Data"] = ctx.Data
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

// ContextProcessor takes any context processor function and returns the
// the resulting string, or "" in the event of a problem with processing
func (t TData) ContextProcessor(fn ctxprc) string {
	if tctx, ok := t["Ctx"].(*Ctx); ok {
		x := fn(tctx)
		return x
	}
	return ""
}

func (t TData) ContextProcessors(ctx *Ctx) {
	if ctx.ctxprocessors != nil {
		for k, v := range ctx.ctxprocessors {
			if v, ok := v.(ctxprc); ok {
				t[k] = t.ContextProcessor(v)
			}
		}
	}
}
