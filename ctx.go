package flotilla

import (
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"

	"github.com/thrisp/engine"
	"github.com/thrisp/flotilla/session"
)

type (
	ResponseWriter interface {
		http.ResponseWriter
		http.Hijacker
		http.Flusher
		http.CloseNotifier

		Status() int
		Size() int
		Written() bool
		WriteHeaderNow()
	}

	// The Current interface handles the information boundary between incoming
	// engine context and flotilla context. The engine must provide a context.Context
	// with a Value fitting this interface.
	Current interface {
		Request() *http.Request
		Data() map[string]interface{}
		Form() url.Values
		Files() map[string][]*multipart.FileHeader
		StatusFunc() (func(int), bool)
		Writer() engine.ResponseWriter
	}

	// A HandlerFunc is any function taking a single parameter, *Ctx
	HandlerFunc func(*Ctx)

	// Ctx is the primary context for passing & setting data between handlerfunc
	// of a route, constructed from the *App and the app engine context data.
	Ctx struct {
		index      int8
		handlers   []HandlerFunc
		deferred   []HandlerFunc
		rw         ResponseWriter
		funcs      map[string]reflect.Value
		processors map[string]reflect.Value
		statusfunc func(int)
		Request    *http.Request
		Session    session.SessionStore
		Data       map[string]interface{}
		App        *App
	}
)

// An adhoc *Ctx built from a responsewriter & a request, not based on a route.
func (a *App) tmpCtx(w ResponseWriter, req *http.Request) *Ctx {
	ctx := &Ctx{App: a,
		Request:    req,
		rw:         w,
		funcs:      reflectFuncs(a.ctxfunctions),
		processors: reflectFuncs(a.ctxprocessors),
	}
	ctx.Start()
	return ctx
}

func (rt Route) newCtx() interface{} {
	return &Ctx{index: -1,
		handlers:   rt.handlers,
		App:        rt.App(),
		Data:       make(map[string]interface{}),
		funcs:      reflectFuncs(rt.CtxFuncs()),
		processors: reflectFuncs(rt.ctxprocessors),
	}
}

func (rt Route) getCtx(c Current) *Ctx {
	ctx := rt.p.Get().(*Ctx)
	ctx.Request = c.Request()
	ctx.rw = c.Writer()
	ctx.Data = c.Data()
	if sf, exists := c.StatusFunc(); exists {
		ctx.statusfunc = sf
	}
	ctx.Start()
	return ctx
}

func (rt Route) putCtx(ctx *Ctx) {
	ctx.index = -1
	ctx.Session = nil
	for k, _ := range ctx.Data {
		delete(ctx.Data, k)
	}
	ctx.deferred = nil
	rt.p.Put(ctx)
}

func (ctx *Ctx) Start() {
	ctx.Session = ctx.App.SessionManager.SessionStart(ctx.rw, ctx.Request)
}

func (ctx *Ctx) Release() {
	if !ctx.rw.Written() {
		ctx.Session.SessionRelease(ctx.rw)
	}
}

// Calls a function with name in *Ctx.funcs passing in the given args.
func (ctx *Ctx) Call(name string, args ...interface{}) (interface{}, error) {
	return call(ctx.funcs[name], args...)
}

// Copies the Ctx with handlers set to nil.
func (ctx *Ctx) Copy() *Ctx {
	var rcopy Ctx = *ctx
	rcopy.index = math.MaxInt8 / 2
	rcopy.handlers = nil
	return &rcopy
}

func (ctx *Ctx) events() {
	ctx.Push(func(c *Ctx) { c.Release() })
	ctx.Next()
	for _, fn := range ctx.deferred {
		fn(ctx)
	}
}

// Executes the pending handlers in the chain inside the calling handlectx.
func (ctx *Ctx) Next() {
	ctx.index++
	s := int8(len(ctx.handlers))
	for ; ctx.index < s; ctx.index++ {
		ctx.handlers[ctx.index](ctx)
	}
}

// Push places a handlerfunc in ctx.deferred for execution after all handlersfuncs have run.
func (ctx *Ctx) Push(fn HandlerFunc) {
	ctx.deferred = append(ctx.deferred, fn)
}

// Sets a new pair key/value in the current Ctx.
func (ctx *Ctx) Set(key string, item interface{}) {
	ctx.Data[key] = item
}

// Get returns the value for the given key or an error if nonexistent.
func (ctx *Ctx) Get(key string) (interface{}, error) {
	item, ok := ctx.Data[key]
	if ok {
		return item, nil
	}
	return nil, newError("Key %s does not exist.", key)
}

// WriteToHeader writes the specified code and values to the response Head.
// values are 2 string arrays indicating the key first and the value second
// to set in the Head.
func (ctx *Ctx) WriteToHeader(code int, values ...[]string) {
	if code >= 0 {
		ctx.rw.WriteHeader(code)
	}
	ctx.ModifyHeader("set", values...)
}

func (ctx *Ctx) ModifyHeader(action string, values ...[]string) {
	switch action {
	case "set":
		for _, v := range values {
			ctx.rw.Header().Set(v[0], v[1])
		}
	default:
		for _, v := range values {
			ctx.rw.Header().Add(v[0], v[1])
		}
	}
}
