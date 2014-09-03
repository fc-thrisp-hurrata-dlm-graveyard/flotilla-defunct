package flotilla

import (
	"errors"
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
	// Use context functions by name and argument
	CtxFunc interface {
		Call(string, ...interface{}) (interface{}, error)
	}

	// Allows passing & setting data between handlers
	Ctx struct {
		rwmem responseWriter
		rw    ResponseWriter
		*D
		ctxfuncs map[string]reflect.Value
		Engine   *Engine
		CtxFunc
	}

	// Ctx data
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
	c := &Ctx{Engine: engine}
	c.rw = &c.rwmem
	c.ctxfuncs = c.Engine.Env.CtxFunctions()
	return c
}

func (engine *Engine) getCtx(w http.ResponseWriter, req *http.Request, params httprouter.Params, handlers []HandlerFunc) *Ctx {
	c := engine.cache.Get().(*Ctx)
	c.rwmem.reset(w)
	c.D = newD(handlers, req, params)
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
	d.ErrorTyped(err, ErrorTypeExternal, meta)
}

func (d *D) ErrorTyped(err error, typ uint32, meta interface{}) {
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

func (c *Ctx) Call(name string, args ...interface{}) (interface{}, error) {
	return call(c.ctxfuncs[name], args...)
}

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

// Forces the system to discontinue calling the pending handlers.
// For example, a handler checks if the request is authorized.
// If not authorized, context.Abort(401) is called and no pending handlers
// called for that request.
func (c *Ctx) Abort(code int) {
	if code >= 0 {
		c.rw.WriteHeader(code)
	}
	c.index = AbortIndex
}

// Fail is the same as Abort plus an error message.
// Calling `context.Fail(500, err)` is equivalent to:
// ```
// context.Error("Operation aborted", err)
// context.Abort(500)
// ```
func (c *Ctx) Fail(code int, err error) {
	c.Error(err, "Operation aborted")
	c.Abort(code)
}

// Sets a new pair key/value just for the specified context.
// It also lazy initializes the hashmap.
func (c *Ctx) Set(key string, item interface{}) {
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[key] = item
}

// Get returns the value for the given key or an error if nonexistent.
func (c *Ctx) Get(key string) (interface{}, error) {
	if c.data != nil {
		item, ok := c.data[key]
		if ok {
			return item, nil
		}
	}
	return nil, newError("Key does not exist.")
}

// MustGet returns the value for the given key or panics if nonexistent.
func (c *Ctx) MustGet(key string) interface{} {
	value, err := c.Get(key)
	if err != nil || value == nil {
		log.Panicf("Key %s doesn't exist", key)
	}
	return value
}

func (c *Ctx) writeHeader(code int, contentType string) {
	if len(contentType) > 0 {
		c.rw.Header().Set("Content-Type", contentType)
	}
	if code >= 0 {
		c.rw.WriteHeader(code)
	}
}

// Returns a HTTP redirect to the specific location.
func redirect(c *Ctx, code int, location string) error {
	if code >= 300 && code <= 308 {
		http.Redirect(c.rw, c.Request, location, code)
		return nil
	} else {
		return newError("Cannot send a redirect with status code %d", code)
	}
}

func (c *Ctx) Redirect(code int, location string) {
	c.Call("redirect", c, code, location)
}

// Writes some data into the body stream and updates the HTTP code.
func servedata(c *Ctx, code int, contentType string, data []byte) error {
	c.writeHeader(code, contentType)
	c.rw.Write(data)
	return nil
}

func (c *Ctx) ServeData(code int, contentType string, data []byte) {
	c.Call("servedata", c, code, contentType, data)
}

// Serves a specified file
func servefile(c *Ctx, f http.File) error {
	fi, err := f.Stat()
	if err == nil {
		http.ServeContent(c.rw, c.Request, fi.Name(), fi.ModTime(), f)
	}
	return err
}

func (c *Ctx) ServeFile(f http.File) {
	c.Call("servefile", c, f)
}

// render & return HTML template
func rendertemplate(c *Ctx, name string, data interface{}) error {
	err := c.Engine.Templator.Render(c.rw, name, data)
	return err
}

func (c *Ctx) RenderTemplate(name string, data interface{}) {
	c.Call("rendertemplate", c, name, data)
}
