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
	}
)

type (
	// Use cross-handler context functions by name and argument
	CtxFunc interface {
		Call(string, ...interface{}) (interface{}, error)
	}

	// The request & response context that allows passing & setting data between handlers
	Ctx struct {
		rwmem responseWriter
		rw    ResponseWriter
		*D
		ctxfuncs map[string]reflect.Value
		engine   *Engine
		session  session.SessionStore
		CtxFunc
	}

	// Specific or dynamic Ctx data from request and for response.
	D struct {
		index    int8
		handlers []HandlerFunc
		Request  *http.Request
		Params   httprouter.Params
		data     map[string]interface{}
		Errors   errorMsgs
	}
)

func (engine *Engine) newCtx() interface{} {
	c := &Ctx{engine: engine}
	c.rw = &c.rwmem
	c.ctxfuncs = c.ctxFunctions()
	return c
}

func (engine *Engine) getCtx(w http.ResponseWriter, req *http.Request, params httprouter.Params, handlers []HandlerFunc) *Ctx {
	c := engine.cache.Get().(*Ctx)
	c.rwmem.reset(w)
	c.D = newD(handlers, req, params)
	c.session = engine.SessionManager.SessionStart(w, req)
	defer c.session.SessionRelease(w)
	return c
}

func newD(handlers []HandlerFunc, req *http.Request, params httprouter.Params) *D {
	d := &D{-1, handlers, req, params, nil, nil}
	return d
}

// Attaches an error that is pushed to a list of errors. It's a good idea
// to call Error for each error that occurred during the resolution of a request.
// A middleware can be used to collect all the errors and push them to a database
// together, print a log, or append it in the HTTP response.
func (d *D) Error(err error, meta interface{}) {
	d.errorTyped(err, ErrorTypeExternal, meta)
}

func (d *D) errorTyped(err error, typ uint32, meta interface{}) {
	d.Errors = append(d.Errors, errorMsg{
		Err:  err.Error(),
		Type: typ,
		Meta: meta,
	})
}

func (d *D) LastError() error {
	s := len(d.Errors)
	if s > 0 {
		return errors.New(d.Errors[s-1].Err)
	} else {
		return nil
	}
}

func (c *Ctx) ctxFunctions() map[string]reflect.Value {
	m := make(map[string]reflect.Value)
	for k, v := range c.engine.Env.ctxfunctions {
		m[k] = valueFunc(v)
	}
	return m
}

// Calls a function with name in Ctx.ctxfuncs passing in the given args.
func (c *Ctx) Call(name string, args ...interface{}) (interface{}, error) {
	return call(c.ctxfuncs[name], args...)
}

// Copies the CTx with handlers set to nil and index AbortIndex
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
	c.Error(err, "Failed.")
	c.Abort(code)
}

// Sets a new pair key/value just for the specified context.
// It also lazy initializes the hashmap.
func (c *Ctx) SetData(key string, item interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[key] = item
}

// Get returns the value for the given key or an error if nonexistent.
func (c *Ctx) GetData(key string) (interface{}, error) {
	if c.data != nil {
		item, ok := c.data[key]
		if ok {
			return item, nil
		}
	}
	return nil, newError("Key %s does not exist.", key)
}

// MustGet returns the value for the given key or panics if nonexistent.
func (c *Ctx) MustGetData(key string) interface{} {
	value, err := c.GetData(key)
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

func servedata(c *Ctx, code int, contentType string, data []byte) error {
	c.WriteHeader(code, contentType)
	c.rw.Write(data)
	return nil
}

// ServeData writes plain data into the body stream and updates the HTTP code,
// using the Ctx servedata function.
func (c *Ctx) ServeData(code int, contentType string, data []byte) {
	c.Call("servedata", c, code, contentType, data)
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

func rendertemplate(c *Ctx, name string, data interface{}) error {
	err := c.engine.Templator.Render(c.rw, name, data)
	return err
}

// Rendertemplate renders an HTML template with the Ctx rendertemplate function
func (c *Ctx) RenderTemplate(name string, data interface{}) {
	c.Call("rendertemplate", c, name, data)
}
