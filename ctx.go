package flotilla

import (
	"log"
	"math"
	"net/http"
	"reflect"

	"github.com/thrisp/engine"
	"github.com/thrisp/flotilla/session"
)

const (
	AbortIndex = math.MaxInt8 / 2
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

type (
	// Ctx is the primary context for passing & setting data between handlerfunc
	// of a route, constructed from the *App and the app engine context data.
	Ctx struct {
		index    int8
		handlers []HandlerFunc
		rw       engine.ResponseWriter
		Request  *http.Request
		RSession session.SessionStore
		RData    ctxdata
		RFunc    ctxfuncs
		App      *App
		Ctx      *engine.Ctx
	}

	// A map as a stash for data in the R.
	ctxdata map[string]interface{}

	// A map of functions used in R
	ctxfuncs map[string]reflect.Value

	// Sent to and accessible within the template, data supplied by the
	// rendertemplate function is set here as Data
	tdata struct {
		Data    interface{}
		Request *http.Request
		Session session.SessionStore
		RData   ctxdata
		Flash   map[string]string
	}
)

// An adhoc *R built from a responsewriter & a request, not based on a route.
func (a *App) tmpCtx(w engine.ResponseWriter, req *http.Request) *Ctx {
	ctx := &Ctx{App: a, Request: req}
	ctx.rw = w
	ctx.RFunc = ctx.ctxFunctions(a.Env)
	ctx.start()
	return ctx
}

func (rt Route) newCtx() interface{} {
	ctx := &Ctx{index: -1,
		handlers: rt.handlers,
		App:      rt.routergroup.app,
		RData:    make(ctxdata),
	}
	ctx.RFunc = ctx.ctxFunctions(rt.routergroup.app.Env)
	return ctx
}

func (rt Route) getCtx(ec *engine.Ctx) *Ctx {
	ctx := rt.cache.Get().(*Ctx)
	ctx.Request = ec.Request
	ctx.rw = ec.RW
	ctx.Ctx = ec
	for _, p := range ec.Params {
		ctx.RData[p.Key] = p.Value
	}
	ctx.start()
	return ctx
}

func (rt Route) putR(ctx *Ctx) {
	ctx.index = -1
	ctx.RSession = nil
	for k, _ := range ctx.RData {
		delete(ctx.RData, k)
	}
	rt.cache.Put(ctx)
}

func (ctx *Ctx) start() {
	ctx.RSession = ctx.App.SessionManager.SessionStart(ctx.rw, ctx.Request)
}

func (ctx *Ctx) release() {
	if ctx.RSession != nil {
		ctx.RSession.SessionRelease(ctx.rw)
	}
}

func (ctx *Ctx) ctxFunctions(e *Env) ctxfuncs {
	m := make(ctxfuncs)
	for k, v := range e.ctxfunctions {
		m[k] = valueFunc(v)
	}
	return m
}

// Calls a function with name in *R.RFuncs passing in the given args.
func (ctx *Ctx) Call(name string, args ...interface{}) (interface{}, error) {
	return call(ctx.RFunc[name], args...)
}

// Copies the Ctx with handlers set to nil and index AbortIndex
func (ctx *Ctx) Copy() *Ctx {
	var rcopy Ctx = *ctx
	rcopy.index = AbortIndex
	rcopy.handlers = nil
	return &rcopy
}

// Executes the pending handlers in the chain inside the calling handlectx.
func (ctx *Ctx) Next() {
	ctx.index++
	s := int8(len(ctx.handlers))
	for ; ctx.index < s; ctx.index++ {
		ctx.handlers[ctx.index](ctx)
	}
}

// Calls Ctx.Status in the Engine, with a fall through to Ctx.Abort.
func (ctx *Ctx) Status(code int) {
	ctx.Ctx.Status(code)
}

// Immediately ends processing of current R and return the code, the same as
// calling *R.Status, but less informative & not configurable.
func (ctx *Ctx) Abort(code int) {
	ctx.Ctx.Abort(code)
}

// Sets a new pair key/value just for the specified context.
func (ctx *Ctx) Set(key string, item interface{}) {
	ctx.RData[key] = item
}

// Get returns the value for the given key or an error if nonexistent.
func (ctx *Ctx) Get(key string) (interface{}, error) {
	item, ok := ctx.RData[key]
	if ok {
		return item, nil
	}
	return nil, newError("Key %s does not exist.", key)
}

// MustGet returns the value for the given key or panics if nonexistent.
func (ctx *Ctx) MustGet(key string) interface{} {
	value, err := ctx.Get(key)
	if err != nil || value == nil {
		log.Panicf("Key %s doesn't exist", key)
	}
	return value
}

// WriteToHeader writes the specified code and content type to the headectx.
func (ctx *Ctx) WriteToHeader(code int, contentType string) {
	if len(contentType) > 0 {
		ctx.rw.Header().Set("Content-Type", contentType)
	}
	if code >= 0 {
		ctx.rw.WriteHeader(code)
	}
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
	ctx.WriteToHeader(code, "text/plain")
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

func templatedata(ctx *Ctx, data interface{}) *tdata {
	return &tdata{
		Data:    data,
		Request: ctx.Request,
		Session: ctx.RSession,
		RData:   ctx.RData,
		Flash:   allflashmessages(ctx),
	}
}

func rendertemplate(ctx *Ctx, name string, data interface{}) error {
	td := templatedata(ctx, data)
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
	if fl := ctx.RSession.Get("_flashes"); fl != nil {
		if fls, ok := fl.(map[string]string); ok {
			fls[category] = message
			ctx.RSession.Set("_flashes", fls)
		}
	} else {
		fl := make(map[string]string)
		fl[category] = message
		ctx.RSession.Set("_flashes", fl)
	}
	return nil
}

// Sets a flash message retrievable from the session.
func (ctx *Ctx) Flash(category string, message string) {
	ctx.Call("flash", ctx, category, message)
}

func flashmessages(ctx *Ctx, categories []string) []string {
	var ret []string
	if fl := ctx.RSession.Get("_flashes"); fl != nil {
		if fls, ok := fl.(map[string]string); ok {
			for k, v := range fls {
				if existsIn(k, categories) {
					ret = append(ret, v)
					delete(fls, k)
				}
			}
			ctx.RSession.Set("_flashes", fls)
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
	if fl := ctx.RSession.Get("_flashes"); fl != nil {
		if fls, ok := fl.(map[string]string); ok {
			ret = fls
		}
	}
	ctx.RSession.Delete("_flashes")
	return ret
}

// Retrieves all flash messages
func (ctx *Ctx) AllFlashMessages() map[string]string {
	ret, _ := ctx.Call("allflashmessages", ctx)
	return ret.(map[string]string)
}
