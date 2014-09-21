package flotilla

import (
	"errors"
	"lcl/flotilla/session"
	"log"
	"math"
	"net/http"
	"reflect"

	"github.com/julienschmidt/httprouter"
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
	CtxFunc interface {
		Call(string, ...interface{}) (interface{}, error)
	}

	// The request & response context that allows passing & setting data
	// between handlers
	Ctx struct {
		index    int8
		handlers []HandlerFunc
		rwmem    responseWriter
		rw       ResponseWriter
		Request  *http.Request
		engine   *Engine
		Session  session.SessionStore
		Errors   errorMsgs
		CtxData  ctxdata
		CtxFunc  ctxfuncs
	}

	// A map of functions used in Ctx
	ctxfuncs map[string]reflect.Value

	// A map as a stash for data in the Ctx. Currently http.Params go here, as
	// well as data set with Ctx.Set().
	ctxdata map[string]interface{}

	// Sent to and accessible within the template, data supplied by the
	// rendertemplate function is set here as Data
	tdata struct {
		Data    interface{}
		Request *http.Request
		Session session.SessionStore
		CtxData ctxdata
	}
)

func (engine *Engine) newCtx() interface{} {
	c := &Ctx{engine: engine}
	c.rw = &c.rwmem
	c.CtxFunc = c.ctxFunctions()
	return c
}

func (engine *Engine) getCtx(w http.ResponseWriter, req *http.Request, params httprouter.Params, handlers []HandlerFunc) *Ctx {
	c := engine.cache.Get().(*Ctx)
	c.rwmem.reset(w)
	c.index = -1
	c.handlers = handlers
	c.Request = req
	c.CtxData = make(ctxdata)
	for _, p := range params {
		c.CtxData[p.Key] = p.Value
	}
	c.Session = engine.SessionManager.SessionStart(w, req)
	defer c.Session.SessionRelease(w)
	return c
}

func (c *Ctx) ctxFunctions() ctxfuncs {
	m := make(ctxfuncs)
	for k, v := range c.engine.Env.ctxfunctions {
		m[k] = valueFunc(v)
	}
	return m
}

// Attaches an error that is pushed to a list of errors. It's a good idea
// to call Error for each error that occurred during the resolution of a request.
// A middleware can be used to collect all the errors and push them to a database
// together, print a log, or append it in the HTTP response.
func (c *Ctx) Error(err error, meta interface{}) {
	c.errorTyped(err, ErrorTypeExternal, meta)
}

func (c *Ctx) errorTyped(err error, typ uint32, meta interface{}) {
	c.Errors = append(c.Errors, errorMsg{
		Err:  err.Error(),
		Type: typ,
		Meta: meta,
	})
}

// Returns the last error for the Ctx.
func (c *Ctx) LastError() error {
	s := len(c.Errors)
	if s > 0 {
		return errors.New(c.Errors[s-1].Err)
	} else {
		return nil
	}
}

// Calls a function with name in Ctx.CtxFuncs passing in the given args.
func (c *Ctx) Call(name string, args ...interface{}) (interface{}, error) {
	return call(c.CtxFunc[name], args...)
}

// Copies the Ctx with handlers set to nil and index AbortIndex
func (c *Ctx) Copy() *Ctx {
	var cp Ctx = *c
	cp.index = AbortIndex
	cp.handlers = nil
	return &cp
}

// Executes the pending handlers in the chain inside the calling handler.
func (c *Ctx) Next() {
	c.index++
	s := int8(len(c.handlers))
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

// Calls an HttpException if available otherwise calls Abort
func (c *Ctx) HttpException(code int) {
	if e, ok := c.engine.HttpExceptions[code]; ok {
		for _, h := range e.handlers {
			h(c)
		}
		c.index = AbortIndex
	} else {
		c.Abort(code)
	}
}

// Immediately ends processing of current Ctx and return the code, the same as
// running c.HttpException, but less informative & not configurable.
func (c *Ctx) Abort(code int) {
	if code >= 0 {
		c.rw.WriteHeader(code)
	}
	c.index = AbortIndex
}

// Fail is the same as Abort plus an error message.
// Calling `c.Fail(500, err)` is equivalent to:
// ```
// c.Error(err, "Failed.")
// c.Abort(500)
// ```
func (c *Ctx) Fail(code int, err error) {
	c.Error(err, err.Error())
	c.Abort(code)
}

// Sets a new pair key/value just for the specified context.
// It also lazy initializes the hashmap.
func (c *Ctx) Set(key string, item interface{}) {
	c.CtxData[key] = item
}

// Get returns the value for the given key or an error if nonexistent.
func (c *Ctx) Get(key string) (interface{}, error) {
	item, ok := c.CtxData[key]
	if ok {
		return item, nil
	}
	return nil, newError("Key %s does not exist.", key)
}

// MustGet returns the value for the given key or panics if nonexistent.
func (c *Ctx) MustGet(key string) interface{} {
	value, err := c.Get(key)
	if err != nil || value == nil {
		log.Panicf("Key %s doesn't exist", key)
	}
	return value
}

// WriteHeader writes the specified code and content type to the header.
func (c *Ctx) WriteHeader(code int, contentType string) {
	if len(contentType) > 0 {
		c.rw.Header().Set("Content-Type", contentType)
	}
	if code >= 0 {
		c.rw.WriteHeader(code)
	}
}

func redirect(c *Ctx, code int, location string) error {
	if code >= 300 && code <= 308 {
		http.Redirect(c.rw, c.Request, location, code)
		return nil
	} else {
		return newError("Cannot send a redirect with status code %d", code)
	}
}

// Returns a HTTP redirect to the specific location, with the specified code.
// using the Ctx redirect function.
func (c *Ctx) Redirect(code int, location string) {
	c.Call("redirect", c, code, location)
}

func servedata(c *Ctx, code int, data []byte) error {
	c.WriteHeader(code, "text/plain")
	c.rw.Write(data)
	return nil
}

// ServeData writes plain data into the body stream and updates the HTTP code,
// using the Ctx servedata function.
func (c *Ctx) ServeData(code int, data []byte) {
	c.Call("servedata", c, code, data)
}

func servefile(c *Ctx, f http.File) error {
	fi, err := f.Stat()
	if err == nil {
		http.ServeContent(c.rw, c.Request, fi.Name(), fi.ModTime(), f)
	}
	return err
}

// ServesFile delivers a specified file using the Ctx servefile function.
func (c *Ctx) ServeFile(f http.File) {
	c.Call("servefile", c, f)
}

func templatedata(c *Ctx, data interface{}) *tdata {
	return &tdata{data, c.Request, c.Session, c.CtxData}
}

func rendertemplate(c *Ctx, name string, data interface{}) error {
	td := templatedata(c, data)
	err := c.engine.Templator.Render(c.rw, name, td)
	return err
}

// RenderTemplate renders an HTML template response with the Ctx rendertemplate
// function.
func (c *Ctx) RenderTemplate(name string, data interface{}) {
	c.Call("rendertemplate", c, name, data)
}

func urlfor(c *Ctx, route string, external bool, params []string) (string, error) {
	if r, ok := c.engine.Routes()[route]; ok {
		route, _ := r.Url(params...)
		if route != nil {
			if external {
				route.Host = c.Request.Host
			}
			return route.String(), nil
		}
	}
	return "", newError("unable to get url for route %s with params %s", route, params)
}

// Provides a relative url for the route specified using the parameters specified,
// using the Ctx urlfor function.
func (c *Ctx) UrlRelative(route string, params ...string) string {
	ret, err := c.Call("urlfor", c, route, false, params)
	if err != nil {
		return err.Error()
	}
	return ret.(string)
}

// Provides a full, external url for the route specified using the given parameters,
// using the Ctx urlfor function.
func (c *Ctx) UrlExternal(route string, params ...string) string {
	ret, err := c.Call("urlfor", c, route, true, params)
	if err != nil {
		return err.Error()
	}
	return ret.(string)
}
