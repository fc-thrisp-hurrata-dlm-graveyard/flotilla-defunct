package flotilla

import (
	"path/filepath"

	"github.com/thrisp/engine"
)

type (
	// A Blueprint gathers any number routes around a prefix and an array of
	// group specific handlers.
	Blueprint struct {
		registered    bool
		deferred      []func()
		app           *App
		children      []*Blueprint
		held          []*Route
		routes        Routes
		group         *engine.Group
		ctxprocessors map[string]interface{}
		Prefix        string
		Handlers      []HandlerFunc
	}
)

// Blueprints provides a flat array of Blueprint instances attached to the App.
func (app *App) Blueprints() []*Blueprint {
	type IterC func(bs []*Blueprint, fn IterC)

	var bps []*Blueprint

	bps = append(bps, app.Blueprint)

	iter := func(bs []*Blueprint, fn IterC) {
		for _, x := range bs {
			bps = append(bps, x)
			fn(x.children, fn)
		}
	}

	iter(app.children, iter)

	return bps
}

func (app *App) RegisterBlueprints(blueprints ...*Blueprint) {
	for _, blueprint := range blueprints {
		if existing, ok := app.existingBlueprint(blueprint.Prefix); ok {
			existing.Use(blueprint.Handlers...)
			app.MergeRoutes(existing, blueprint.routes)
		} else {
			app.children = append(app.children, blueprint)
			blueprint.Register(app)
		}
	}
}

func (app *App) existingBlueprint(prefix string) (*Blueprint, bool) {
	for _, b := range app.Blueprints() {
		if b.Prefix == prefix {
			return b, true
		}
	}
	return nil, false
}

// Mount takes an unregistered blueprint, registering and mounting the routes to
// the provided string mount point with a copy of the blueprint. If inherit is
// true, the blueprint becomes a child blueprint of app.Blueprint.
func (app *App) Mount(mount string, inherit bool, blueprint *Blueprint) error {
	if blueprint.registered {
		return newError("only unregistered blueprints may be mounted; %s is already registered", blueprint.Prefix)
	}
	var mountblueprint *Blueprint
	newprefix := filepath.ToSlash(filepath.Join(mount, blueprint.Prefix))
	if inherit {
		mountblueprint = app.New(newprefix)
	} else {
		mountblueprint = NewBlueprint(newprefix)
	}
	for _, route := range blueprint.held {
		mountblueprint.Handle(route)
	}
	app.RegisterBlueprints(mountblueprint)
	return nil
}

func (b *Blueprint) pathFor(path string) string {
	joined := filepath.ToSlash(filepath.Join(b.Prefix, path))
	// Append a '/' if the last component had one, but only if it's not there already
	if len(path) > 0 && path[len(path)-1] == '/' && joined[len(joined)-1] != '/' {
		return joined + "/"
	}
	return joined
}

// NewBlueprint returns a new Blueprint with the provided string prefix.
func NewBlueprint(prefix string) *Blueprint {
	return &Blueprint{Prefix: prefix,
		routes:        make(Routes),
		ctxprocessors: make(map[string]interface{}),
	}
}

// RegisteredBlueprint creates a new Blueprint and registers it with the App.
func RegisteredBlueprint(prefix string, app *App) *Blueprint {
	b := NewBlueprint(prefix)
	b.Register(app)
	return b
}

func (b *Blueprint) Register(a *App) {
	b.app = a
	b.group = a.engine.Group.New(b.Prefix)
	b.rundeferred()
	b.registered = true
}

func (b *Blueprint) rundeferred() {
	for _, fn := range b.deferred {
		fn()
	}
	b.deferred = nil
}

// New Creates a new child Blueprint from the existing Blueprint.
func (b *Blueprint) New(component string, handlers ...HandlerFunc) *Blueprint {
	prefix := b.pathFor(component)

	newb := RegisteredBlueprint(prefix, b.app)
	newb.ctxprocessors = b.ctxprocessors
	newb.Handlers = b.combineHandlers(handlers)

	b.children = append(b.children, newb)

	return newb
}

func (b *Blueprint) combineHandlers(handlers []HandlerFunc) []HandlerFunc {
	s := len(b.Handlers) + len(handlers)
	h := make([]HandlerFunc, 0, s)
	h = append(h, b.Handlers...)
	h = append(h, handlers...)
	return h
}

func (b *Blueprint) handlerExists(outside HandlerFunc) bool {
	for _, inside := range b.Handlers {
		if equalFunc(inside, outside) {
			return true
		}
	}
	return false
}

// Use adds any number of HandlerFunc to the Blueprint which will be run before
// route handlers for all Route attached to the Blueprint.
func (b *Blueprint) Use(handlers ...HandlerFunc) {
	for _, handler := range handlers {
		if !b.handlerExists(handler) {
			b.Handlers = append(b.Handlers, handler)
		}
	}
}

// UseAt adds any number of HandlerFunc to the Blueprint as middleware when you
// must control the position in relation to other middleware.
func (b *Blueprint) UseAt(index int, handlers ...HandlerFunc) {
	if index > len(b.Handlers) {
		b.Use(handlers...)
		return
	}

	var newh []HandlerFunc

	for _, handler := range handlers {
		if !b.handlerExists(handler) {
			newh = append(newh, handler)
		}
	}

	before := b.Handlers[:index]
	after := append(newh, b.Handlers[index:]...)
	b.Handlers = append(before, after...)
}

func (b *Blueprint) Add(r *Route) {
	if r.Name != "" {
		b.routes[r.Name] = r
	} else {
		b.routes[r.Named()] = r
	}
}

func (b *Blueprint) Hold(r *Route) {
	b.held = append(b.held, r)
}

func (b *Blueprint) Push(register func(), route *Route) {
	if b.registered {
		register()
	} else {
		if route != nil {
			b.Hold(route)
		}
		b.deferred = append(b.deferred, register)
	}
}

func (b *Blueprint) propagate(name string, fn interface{}) {
	for _, blueprint := range b.children {
		blueprint.CtxProcessor(name, fn)
	}
	for _, rt := range b.routes {
		rt.CtxProcessor(name, fn)
	}
}

func (b *Blueprint) CtxProcessor(name string, fn interface{}) {
	b.ctxprocessors[name] = fn
	// update existing blueprints & routes
	b.propagate(name, fn)
}

func (b *Blueprint) CtxProcessors(cp map[string]interface{}) {
	for k, v := range cp {
		b.CtxProcessor(k, v)
	}
}

func (b *Blueprint) onregister(route *Route) {
	route.blueprint = b
	route.handlers = b.combineHandlers(route.handlers)
	route.CtxProcessors(b.ctxprocessors)
	route.path = b.pathFor(route.base)
	route.p.New = route.newCtx
	route.registered = true
}

// Handle registers new handlers and/or handlers with a constructed Route.
// method. For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used by specifying path & handlers.
func (b *Blueprint) Handle(route *Route) {
	register := func() {
		b.onregister(route)
		b.Add(route)
		b.group.Handle(route.base, route.method, route.handle)
	}
	b.Push(register, route)
}

func (b *Blueprint) POST(path string, handlers ...HandlerFunc) {
	b.Handle(NewRoute("POST", path, false, handlers))
}

func (b *Blueprint) GET(path string, handlers ...HandlerFunc) {
	b.Handle(NewRoute("GET", path, false, handlers))
}

func (b *Blueprint) DELETE(path string, handlers ...HandlerFunc) {
	b.Handle(NewRoute("DELETE", path, false, handlers))
}

func (b *Blueprint) PATCH(path string, handlers ...HandlerFunc) {
	b.Handle(NewRoute("PATCH", path, false, handlers))
}

func (b *Blueprint) PUT(path string, handlers ...HandlerFunc) {
	b.Handle(NewRoute("PUT", path, false, handlers))
}

func (b *Blueprint) OPTIONS(path string, handlers ...HandlerFunc) {
	b.Handle(NewRoute("OPTIONS", path, false, handlers))
}

func (b *Blueprint) HEAD(path string, handlers ...HandlerFunc) {
	b.Handle(NewRoute("HEAD", path, false, handlers))
}

// STATIC adds a Static route handled by the app engine, based on the group prefix.
func (b *Blueprint) STATIC(path string) {
	b.app.StaticDirs(dropTrailing(path, "*filepath"))
	b.Handle(NewRoute("GET", path, true, []HandlerFunc{handleStatic}))
}

// Custom HttpStatus for the group, set and called from engine HttpStatuses
func (b *Blueprint) StatusHandle(code int, handlers ...HandlerFunc) {
	statushandler := func(c *engine.Ctx) {
		statusCtx := b.app.tmpCtx(c.RW, c.Request)
		s := len(handlers)
		for i := 0; i < s; i++ {
			handlers[i](statusCtx)
		}
	}
	register := func() {
		if ss, ok := b.group.HttpStatuses[code]; ok {
			ss.Update(statushandler)
		} else {
			ns := engine.NewHttpStatus(code, string(code))
			ns.Update(statushandler)
			b.group.HttpStatuses.New(ns)
		}
	}
	b.Push(register, nil)
}
