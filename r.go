package flotilla

import (
	"fmt"
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
		"redirect":       redirect,
		"servedata":      servedata,
		"servefile":      servefile,
		"rendertemplate": rendertemplate,
		"urlfor":         urlfor,
	}
)

type (
	// Use cross-handler context functions by name and argument
	RFunc interface {
		Call(string, ...interface{}) (interface{}, error)
	}

	// R is the primary context for passing & setting data between handlerfunc
	// of a route, constructed from the *App and the app engine context data.
	R struct {
		index      int8
		handlers   []HandlerFunc
		rw         ResponseWriter
		Request    *http.Request
		RSession   session.SessionStore
		RData      ctxdata
		RFunc      ctxfuncs
		app        *App
		httpstatus HttpStatuses
		Ctx        *engine.Ctx
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
	}
)

func (a *App) tmpR(w http.ResponseWriter, req *http.Request) *R {
	r := &R{Request: req}
	rw := &responseWriter{}
	rw.reset(w)
	r.rw = rw
	r.RFunc = r.ctxFunctions(a.Env)
	return r
}

func (rt Route) newR() interface{} {
	r := &R{index: -1,
		handlers:   rt.handlers,
		app:        rt.routergroup.app,
		httpstatus: rt.routergroup.HttpStatuses,
		RData:      make(ctxdata),
	}
	r.RFunc = r.ctxFunctions(rt.routergroup.app.Env)
	return r
}

func (rt Route) getR(c *engine.Ctx) *R {
	r := rt.cache.Get().(*R)
	r.Request = c.Request
	r.rw = c.RW
	r.Ctx = c
	for _, p := range c.Params {
		r.RData[p.Key] = p.Value
	}
	r.start()
	//r.RSession = r.app.SessionManager.SessionStart(r.rw, r.Request)
	return r
}

func (rt Route) putR(r *R) {
	r.index = -1
	r.RSession = nil
	for k, _ := range r.RData {
		delete(r.RData, k)
	}
	rt.cache.Put(r)
}

func (r *R) start() {
	r.RSession = r.app.SessionManager.SessionStart(r.rw, r.Request)
}

func (r *R) release() {
	r.RSession.SessionRelease(r.rw)
}

func (r *R) ctxFunctions(e *Env) ctxfuncs {
	m := make(ctxfuncs)
	for k, v := range e.ctxfunctions {
		m[k] = valueFunc(v)
	}
	return m
}

// Calls a function with name in *R.RFuncs passing in the given args.
func (r *R) Call(name string, args ...interface{}) (interface{}, error) {
	return call(r.RFunc[name], args...)
}

// Copies the Ctx with handlers set to nil and index AbortIndex
func (r *R) Copy() *R {
	var rcopy R = *r
	rcopy.index = AbortIndex
	rcopy.handlers = nil
	return &rcopy
}

// Executes the pending handlers in the chain inside the calling handler.
func (r *R) Next() {
	r.index++
	s := int8(len(r.handlers))
	for ; r.index < s; r.index++ {
		r.handlers[r.index](r)
	}
}

// Calls HandlerFunc from a group HttpStatuses attached to *R, if available
// otherwise calls Ctx.Status with a fall through to Ctx.Abort in the Engine.
func (r *R) Status(code int) {
	fmt.Printf("%+v\n", r)
	if handlers, ok := r.httpstatus[code]; ok {
		s := len(handlers)
		for i := 0; i < s; i++ {
			handlers[i](r)
		}
	} else {
		r.Ctx.Status(code)
	}
}

// Immediately ends processing of current R and return the code, the same as
// running r.HttpException, but less informative & not configurable.
func (r *R) Abort(code int) {
	r.Ctx.Abort(code)
}

// Sets a new pair key/value just for the specified context.
// It also lazy initializes the hashmap.
func (r *R) Set(key string, item interface{}) {
	r.RData[key] = item
}

// Get returns the value for the given key or an error if nonexistent.
func (r *R) Get(key string) (interface{}, error) {
	item, ok := r.RData[key]
	if ok {
		return item, nil
	}
	return nil, newError("Key %s does not exist.", key)
}

// MustGet returns the value for the given key or panics if nonexistent.
func (r *R) MustGet(key string) interface{} {
	value, err := r.Get(key)
	if err != nil || value == nil {
		log.Panicf("Key %s doesn't exist", key)
	}
	return value
}

// WriteHeader writes the specified code and content type to the header.
func (r *R) WriteHeader(code int, contentType string) {
	if len(contentType) > 0 {
		r.rw.Header().Set("Content-Type", contentType)
	}
	if code >= 0 {
		r.rw.WriteHeader(code)
	}
}

func redirect(r *R, code int, location string) error {
	if code >= 300 && code <= 308 {
		http.Redirect(r.rw, r.Request, location, code)
		return nil
	} else {
		return newError("Cannot send a redirect with status code %d", code)
	}
}

// Returns a HTTP redirect to the specific location, with the specified code.
// using the Ctx redirect function.
func (r *R) Redirect(code int, location string) {
	r.Call("redirect", r, code, location)
}

func servedata(r *R, code int, data []byte) error {
	r.release()
	r.WriteHeader(code, "text/plain")
	r.rw.Write(data)
	return nil
}

// ServeData writes plain data into the body stream and updates the HTTP code,
// using the Ctx servedata function.
func (r *R) ServeData(code int, data []byte) {
	r.Call("servedata", r, code, data)
}

func servefile(r *R, f http.File) error {
	fi, err := f.Stat()
	if err == nil {
		http.ServeContent(r.rw, r.Request, fi.Name(), fi.ModTime(), f)
	}
	return err
}

// ServesFile delivers a specified file using the Ctx servefile function.
func (r *R) ServeFile(f http.File) {
	r.Call("servefile", r, f)
}

func templatedata(r *R, data interface{}) *tdata {
	return &tdata{data, r.Request, r.RSession, r.RData}
}

func rendertemplate(r *R, name string, data interface{}) error {
	td := templatedata(r, data)
	r.release()
	err := r.app.Templator.Render(r.rw, name, td)
	return err
}

// RenderTemplate renders an HTML template response with the R rendertemplate
// function.
func (r *R) RenderTemplate(name string, data interface{}) {
	r.Call("rendertemplate", r, name, data)
}

func urlfor(r *R, route string, external bool, params []string) (string, error) {
	if route, ok := r.app.Routes()[route]; ok {
		routeurl, _ := route.Url(params...)
		if routeurl != nil {
			if external {
				routeurl.Host = r.Request.Host
			}
			return routeurl.String(), nil
		}
	}
	return "", newError("unable to get url for route %s with params %s", route, params)
}

// Provides a relative url for the route specified using the parameters specified,
// using the R urlfor function.
func (r *R) UrlRelative(route string, params ...string) string {
	ret, err := r.Call("urlfor", r, route, false, params)
	if err != nil {
		return err.Error()
	}
	return ret.(string)
}

// Provides a full, external url for the route specified using the given parameters,
// using the R urlfor function.
func (r *R) UrlExternal(route string, params ...string) string {
	ret, err := r.Call("urlfor", r, route, true, params)
	if err != nil {
		return err.Error()
	}
	return ret.(string)
}
