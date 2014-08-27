package flotilla

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

const (
	AbortIndex = math.MaxInt8 / 2
)

type (
	// Allows us to pass variables between middleware & manage the flow
	Ctx struct {
		writermem responseWriter
		Request   *http.Request
		Writer    ResponseWriter
		Keys      map[string]interface{}
		Errors    errorMsgs
		Params    httprouter.Params
		Engine    *Engine
		handlers  []HandlerFunc
		index     int8
	}
)

func (engine *Engine) createCtx(w http.ResponseWriter, req *http.Request, params httprouter.Params, handlers []HandlerFunc) *Ctx {
	c := engine.cache.Get().(*Ctx)
	c.writermem.reset(w)
	c.Request = req
	c.Params = params
	c.handlers = handlers
	c.Keys = nil
	c.index = -1
	c.Errors = c.Errors[0:0]
	return c
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
		c.Writer.WriteHeader(code)
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

func (c *Ctx) ErrorTyped(err error, typ uint32, meta interface{}) {
	c.Errors = append(c.Errors, errorMsg{
		Err:  err.Error(),
		Type: typ,
		Meta: meta,
	})
}

// Attaches an error to the current context. The error is pushed to a list of
// errors. It's a good idea to call Error for each error that occurred during
// the resolution of a request. A middleware can be used to collect all the
// errors and push them to a database together, print a log, or append it in
// the HTTP response.
func (c *Ctx) Error(err error, meta interface{}) {
	c.ErrorTyped(err, ErrorTypeExternal, meta)
}

func (c *Ctx) LastError() error {
	s := len(c.Errors)
	if s > 0 {
		return errors.New(c.Errors[s-1].Err)
	} else {
		return nil
	}
}

// Sets a new pair key/value just for the specified context.
// It also lazy initializes the hashmap.
func (c *Ctx) Set(key string, item interface{}) {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = item
}

// Get returns the value for the given key or an error if nonexistent.
func (c *Ctx) Get(key string) (interface{}, error) {
	if c.Keys != nil {
		item, ok := c.Keys[key]
		if ok {
			return item, nil
		}
	}
	return nil, errors.New("Key does not exist.")
}

// MustGet returns the value for the given key or panics if nonexistent.
func (c *Ctx) MustGet(key string) interface{} {
	value, err := c.Get(key)
	if err != nil || value == nil {
		log.Panicf("Key %s doesn't exist", key)
	}
	return value
}

// Returns a HTTP redirect to the specific location.
func (c *Ctx) Redirect(code int, location string) {
	if code >= 300 && code <= 308 {
		http.Redirect(c.Writer, c.Request, location, code)
	} else {
		panic(fmt.Sprintf("Cannot send a redirect with status code %d", code))
	}
}

func (c *Ctx) writeHeader(code int, contentType string) {
	if len(contentType) > 0 {
		c.Writer.Header().Set("Content-Type", contentType)
	}
	if code >= 0 {
		c.Writer.WriteHeader(code)
	}
}

// Writes some data into the body stream and updates the HTTP code.
func (c *Ctx) Data(code int, contentType string, data []byte) {
	c.writeHeader(code, contentType)
	c.Writer.Write(data)
}

// Writes the specified file into the body stream
func (c *Ctx) ReturnFile(filepath string) {
	http.ServeFile(c.Writer, c.Request, filepath)
}

// Serves a specified file
func (c *Ctx) ServeFile(f http.File) {
	fi, err := f.Stat()
	if err == nil {
		http.ServeContent(c.Writer, c.Request, fi.Name(), fi.ModTime(), f)
	}
}

func (c *Ctx) RenderTemplate(name string, data interface{}) {
	c.writeHeader(200, "text/html")
	c.Engine.Templator.Render(c.Writer, name, data)
}
