package flotilla

import (
	"net/http"
)

var (
	builtinctxfuncs = map[string]interface{}{
		"allflashmessages": allflashmessages,
		"flash":            flash,
		"flashmessages":    flashmessages,
		"redirect":         redirect,
		"rendertemplate":   rendertemplate,
		"servedata":        servedata,
		"servefile":        servefile,
		"urlfor":           urlfor,
	}
)

func validctxfunc(fn interface{}) error {
	if goodFunc(valueFunc(fn).Type()) {
		return nil
	} else {
		return newError("function %q is not a valid CtxFunc; must be a function and return must be 1 value, or 1 value and 1 error value", fn)
	}
}

func makectxfuncs(e *Env) ctxfuncs {
	m := make(ctxfuncs)
	for k, v := range e.ctxfunctions {
		m[k] = valueFunc(v)
	}
	return m
}

func redirect(ctx *Ctx, code int, location string) error {
	if code >= 300 && code <= 308 {
		http.Redirect(ctx.rw, ctx.Request, location, code)
		ctx.release()
		ctx.rw.WriteHeaderNow()
		return nil
	} else {
		return newError("Cannot send a redirect with status code %d", code)
	}
}

// Returns a HTTP redirect to the specific location, with the specified code.
// using the Ctx redirect function.
func (ctx *Ctx) Redirect(code int, location string) {
	ctx.Call("redirect", ctx, code, location)
}

func servedata(ctx *Ctx, code int, data []byte) error {
	ctx.release()
	ctx.WriteToHeader(code, []string{"Content-Type", "text/plain"})
	ctx.rw.Write(data)
	return nil
}

// ServeData writes plain data into the body stream and updates the HTTP code,
// using the Ctx servedata function.
func (ctx *Ctx) ServeData(code int, data []byte) {
	ctx.Call("servedata", ctx, code, data)
}

func servefile(ctx *Ctx, f http.File) error {
	ctx.release()
	fi, err := f.Stat()
	if err == nil {
		http.ServeContent(ctx.rw, ctx.Request, fi.Name(), fi.ModTime(), f)
	}
	return err
}

// ServesFile delivers a specified file using the Ctx servefile function.
func (ctx *Ctx) ServeFile(f http.File) {
	ctx.Call("servefile", ctx, f)
}

func rendertemplate(ctx *Ctx, name string, data interface{}) error {
	td := TemplateData(ctx, data)
	ctx.release()
	err := ctx.App.Templator.Render(ctx.rw, name, td)
	return err
}

// RenderTemplate renders an HTML template response with the R rendertemplate
// function.
func (ctx *Ctx) RenderTemplate(name string, data interface{}) {
	ctx.Call("rendertemplate", ctx, name, data)
}

func urlfor(ctx *Ctx, route string, external bool, params []string) (string, error) {
	if route, ok := ctx.App.Routes()[route]; ok {
		routeurl, _ := route.Url(params...)
		if routeurl != nil {
			if external {
				routeurl.Host = ctx.Request.Host
			}
			return routeurl.String(), nil
		}
	}
	return "", newError("unable to get url for route %s with params %s", route, params)
}

// Provides a relative url for the route specified using the parameters specified,
// using the R urlfor function.
func (ctx *Ctx) UrlRelative(route string, params ...string) string {
	ret, err := ctx.Call("urlfor", ctx, route, false, params)
	if err != nil {
		return err.Error()
	}
	return ret.(string)
}

// Provides a full, external url for the route specified using the given parameters,
// using the R urlfor function.
func (ctx *Ctx) UrlExternal(route string, params ...string) string {
	ret, err := ctx.Call("urlfor", ctx, route, true, params)
	if err != nil {
		return err.Error()
	}
	return ret.(string)
}

func flash(ctx *Ctx, category string, message string) error {
	if fl := ctx.Session.Get("_flashes"); fl != nil {
		if fls, ok := fl.(map[string]string); ok {
			fls[category] = message
			ctx.Session.Set("_flashes", fls)
		}
	} else {
		fl := make(map[string]string)
		fl[category] = message
		ctx.Session.Set("_flashes", fl)
	}
	return nil
}

// Sets a flash message retrievable from the session.
func (ctx *Ctx) Flash(category string, message string) {
	ctx.Call("flash", ctx, category, message)
}

func flashmessages(ctx *Ctx, categories []string) []string {
	var ret []string
	if fl := ctx.Session.Get("_flashes"); fl != nil {
		if fls, ok := fl.(map[string]string); ok {
			for k, v := range fls {
				if existsIn(k, categories) {
					ret = append(ret, v)
					delete(fls, k)
				}
			}
			ctx.Session.Set("_flashes", fls)
		}
	}
	return ret
}

// Gets flash messages set in the session by provided categories, deleting those
// returned from the session.
func (ctx *Ctx) FlashMessages(categories ...string) []string {
	ret, _ := ctx.Call("flashmessages", ctx, categories)
	return ret.([]string)
}

func allflashmessages(ctx *Ctx) map[string]string {
	var ret map[string]string
	if fl := ctx.Session.Get("_flashes"); fl != nil {
		if fls, ok := fl.(map[string]string); ok {
			ret = fls
		}
	}
	ctx.Session.Delete("_flashes")
	return ret
}

// Retrieves all flash messages
func (ctx *Ctx) AllFlashMessages() map[string]string {
	ret, _ := ctx.Call("allflashmessages", ctx)
	return ret.(map[string]string)
}
