package fleet

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

const (
	AbortIndex        = math.MaxInt8 / 2
	ErrorTypeInternal = 1 << iota
	ErrorTypeExternal = 1 << iota
	ErrorTypeAll      = 0xffffffff
)

// Used internally to collect errors that occurred during an http request.
type errorMsg struct {
	Err  string      `json:"error"`
	Type uint32      `json:"-"`
	Meta interface{} `json:"meta"`
}

type errorMsgs []errorMsg

func (a errorMsgs) ByType(typ uint32) errorMsgs {
	if len(a) == 0 {
		return a
	}
	result := make(errorMsgs, 0, len(a))
	for _, msg := range a {
		if msg.Type&typ > 0 {
			result = append(result, msg)
		}
	}
	return result
}

func (a errorMsgs) String() string {
	if len(a) == 0 {
		return ""
	}
	var buffer bytes.Buffer
	for i, msg := range a {
		text := fmt.Sprintf("Error #%02d: %s \n     Meta: %v\n", (i + 1), msg.Err, msg.Meta)
		buffer.WriteString(text)
	}
	return buffer.String()
}

// Allows us to pass variables between middleware & manage the flow
type Context struct {
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

func (engine *Engine) createContext(w http.ResponseWriter, req *http.Request, params httprouter.Params, handlers []HandlerFunc) *Context {
	c := engine.cache.Get().(*Context)
	c.writermem.reset(w)
	c.Request = req
	c.Params = params
	c.handlers = handlers
	c.Keys = nil
	c.index = -1
	c.Errors = c.Errors[0:0]
	return c
}

func (c *Context) Copy() *Context {
	var cp Context = *c
	cp.index = AbortIndex
	cp.handlers = nil
	return &cp
}

// Next should be used only in the middlewares.
// It executes the pending handlers in the chain inside the calling handler.
func (c *Context) Next() {
	c.index++
	s := int8(len(c.handlers))
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

// Forces the system to do not continue calling the pending handlers.
// For example, the first handler checks if the request is authorized. If it's not, context.Abort(401) should be called.
// The rest of pending handlers would never be called for that request.
func (c *Context) Abort(code int) {
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
func (c *Context) Fail(code int, err error) {
	c.Error(err, "Operation aborted")
	c.Abort(code)
}

func (c *Context) ErrorTyped(err error, typ uint32, meta interface{}) {
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
func (c *Context) Error(err error, meta interface{}) {
	c.ErrorTyped(err, ErrorTypeExternal, meta)
}

func (c *Context) LastError() error {
	s := len(c.Errors)
	if s > 0 {
		return errors.New(c.Errors[s-1].Err)
	} else {
		return nil
	}
}

// Sets a new pair key/value just for the specified context.
// It also lazy initializes the hashmap.
func (c *Context) Set(key string, item interface{}) {
	if c.Keys == nil {
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = item
}

// Get returns the value for the given key or an error if nonexistent.
func (c *Context) Get(key string) (interface{}, error) {
	if c.Keys != nil {
		item, ok := c.Keys[key]
		if ok {
			return item, nil
		}
	}
	return nil, errors.New("Key does not exist.")
}

// MustGet returns the value for the given key or panics if nonexistent.
func (c *Context) MustGet(key string) interface{} {
	value, err := c.Get(key)
	if err != nil || value == nil {
		log.Panicf("Key %s doesn't exist", key)
	}
	return value
}

// Writes some data into the body stream and updates the HTTP code.
func (c *Context) Data(code int, contentType string, data []byte) {
	if len(contentType) > 0 {
		c.Writer.Header().Set("Content-Type", contentType)
	}
	if code >= 0 {
		c.Writer.WriteHeader(code)
	}
	c.Writer.Write(data)
}

// Writes the specified file into the body stream
func (c *Context) File(filepath string) {
	http.ServeFile(c.Writer, c.Request, filepath)
}
