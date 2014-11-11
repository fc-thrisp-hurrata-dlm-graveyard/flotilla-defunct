package flotilla

import (
	"math"
	"net/http"
	"reflect"

	"github.com/thrisp/engine"
	"github.com/thrisp/flotilla/session"
)

type (
	// Ctx is the primary context for passing & setting data between handlerfunc
	// of a route, constructed from the *App and the app engine context data.
	Ctx struct {
		index    int8
		handlers []HandlerFunc
		rw       engine.ResponseWriter
		Request  *http.Request
		Session  session.SessionStore
		Data     map[string]interface{}
		App      *App
		Ctx      *engine.Ctx
		ctxfuncs map[string]reflect.Value
		ctxprcss map[string]reflect.Value
	}
)

// An adhoc *Ctx built from a responsewriter & a request, not based on a route.
func (a *App) tmpCtx(w engine.ResponseWriter, req *http.Request) *Ctx {
	ctx := &Ctx{App: a,
		Request:  req,
		rw:       w,
		ctxfuncs: reflectFuncs(a.Env.ctxfunctions),
		ctxprcss: reflectFuncs(a.ctxprcss),
	}
	ctx.start()
	return ctx
}

func (rt Route) newCtx() interface{} {
	return &Ctx{index: -1,
		handlers: rt.handlers,
		App:      rt.routergroup.app,
		Data:     make(map[string]interface{}),
		ctxfuncs: reflectFuncs(rt.routergroup.app.Env.ctxfunctions),
		ctxprcss: reflectFuncs(rt.ctxprcss),
	}
}

func (rt Route) getCtx(ec *engine.Ctx) *Ctx {
	ctx := rt.p.Get().(*Ctx)
	ctx.Request = ec.Request
	ctx.rw = ec.RW
	ctx.Ctx = ec
	for _, p := range ec.Params {
		ctx.Data[p.Key] = p.Value
	}
	ctx.start()
	return ctx
}

func (rt Route) putR(ctx *Ctx) {
	ctx.index = -1
	ctx.Session = nil
	for k, _ := range ctx.Data {
		delete(ctx.Data, k)
	}
	rt.p.Put(ctx)
}

func (ctx *Ctx) start() {
	ctx.Session = ctx.App.SessionManager.SessionStart(ctx.rw, ctx.Request)
}

func (ctx *Ctx) release() {
	ctx.Session.SessionRelease(ctx.rw)
}

// Calls a function with name in *Ctx.ctxfuncs passing in the given args.
func (ctx *Ctx) Call(name string, args ...interface{}) (interface{}, error) {
	return call(ctx.ctxfuncs[name], args...)
}

// Copies the Ctx with handlers set to nil; useful for read only copies in goroutines.
func (ctx *Ctx) Copy() *Ctx {
	var rcopy Ctx = *ctx
	rcopy.index = math.MaxInt8 / 2
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

// Immediately ends processing of current Ctx and return the code, the same as
// calling *Ctx.Status, but less informative & not configurable.
func (ctx *Ctx) Abort(code int) {
	ctx.Ctx.Abort(code)
}

// Sets a new pair key/value just for the specified context.
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
	for _, v := range values {
		ctx.rw.Header().Set(v[0], v[1])
	}
}
