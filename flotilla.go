package flotilla

import (
	"net/http"

	"sync"

	"github.com/julienschmidt/httprouter"
)

type (
	HandlerFunc func(*Ctx)

	// The base of running a Flotilla instance is an Engine with a Name, an Env
	// with information specific to running the engine, and a chain of RouterGroups
	Engine struct {
		Name string
		*Env
		*RouterGroup
		cache  sync.Pool
		router *httprouter.Router
	}

	// Essential information about an engine for export to another engine
	Blueprint struct {
		Name   string
		Prefix string
		Groups []*RouterGroup
		Env    *Env
	}

	// Engine extension interface
	Flotilla interface {
		Blueprint() *Blueprint
	}
)

// Returns an empty engine instance
func Empty() *Engine {
	return &Engine{}
}

// Returns a new engine
func New(name string) *Engine {
	engine := &Engine{Name: name,
		Env:    BaseEnv(),
		router: httprouter.New(),
	}
	engine.RouterGroup = &RouterGroup{prefix: "/", engine: engine}
	engine.router.NotFound = engine.default404
	engine.router.PanicHandler = engine.default500
	engine.cache.New = engine.newCtx
	return engine
}

// Returns a basic engine instance with sensible defaults
func Basic() *Engine {
	engine := New("flotilla")
	engine.Use(Logger())
	engine.Static("static")
	return engine
}

// Extends an engine with Flotilla interface
func (engine *Engine) Extend(f Flotilla) {
	blueprint := f.Blueprint()
	if engine.Env.flotilla == nil {
		engine.Env.flotilla = make(map[string]Flotilla)
	}
	engine.Env.flotilla[blueprint.Name] = f
	engine.MergeRouterGroups(blueprint.Groups)
	engine.Env.MergeEnv(blueprint.Env)
}

// Middleware handlers for the engine
func (engine *Engine) Use(middlewares ...HandlerFunc) {
	engine.RouterGroup.Use(middlewares...)
	engine.finalNoRoute = engine.combineHandlers(engine.noRoute)
}

// Methods to ensure the engine satisfies interface Flotilla
func (engine *Engine) Blueprint() *Blueprint {
	return &Blueprint{Name: engine.Name,
		Groups: engine.Groups(),
		Env:    engine.Env}
}

// A slice of all RouterGroup instances attached to the engine
func (engine *Engine) Groups() []*RouterGroup {
	type IterC func(r []*RouterGroup, fn IterC)

	var rg []*RouterGroup

	rg = append(rg, engine.RouterGroup)

	iter := func(r []*RouterGroup, fn IterC) {
		for _, x := range r {
			rg = append(rg, x)
			fn(x.children, fn)
		}
	}

	iter(engine.children, iter)

	return rg
}

// A slice of Route, with all engine routes from all engine routergroups
func (engine *Engine) Routes() []*Route {
	var allroutes []*Route
	for _, group := range engine.Groups() {
		for _, route := range group.routes {
			allroutes = append(allroutes, route)
		}
	}
	return allroutes
}

// Merges a slice of RouterGroup instances into the engine
func (engine *Engine) MergeRouterGroups(groups []*RouterGroup) {
	for _, x := range groups {
		if group, ok := engine.existingGroup(x.prefix); ok {
			group.Use(x.Handlers...)
			engine.MergeRoutes(group, x.routes)
		} else {
			newgroup := engine.RouterGroup.NewGroup(x.prefix, x.Handlers...)
			engine.MergeRoutes(newgroup, x.routes)
		}
	}
}

// Returns group & boolean group existence by the given prefix from engine routergroups
func (engine *Engine) existingGroup(prefix string) (*RouterGroup, bool) {
	for _, g := range engine.Groups() {
		if g.prefix == prefix {
			return g, true
		}
	}
	return nil, false
}

// Returns boolean existence of a route in relation to engine routes, based on
// visiblepath of route
func (engine *Engine) existingRoute(route *Route) bool {
	for _, r := range engine.Routes() {
		if route.visiblepath == r.visiblepath {
			return true
		}
	}
	return false
}

// Merges the given group with the given routes using existence of route in engine
func (engine *Engine) MergeRoutes(group *RouterGroup, routes []*Route) {
	for _, route := range routes {
		if route.static && !engine.existingRoute(route) {
			group.Static(route.staticpath)
		}
		if !route.static && !engine.existingRoute(route) {
			group.Handle(route)
		}
	}
}

func (engine *Engine) Init() {
	engine.parseFlags()
	engine.Env.SessionInit()
}

// ServeHTTP makes the router implement the http.Handler interface.
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	engine.router.ServeHTTP(w, req)
}

func (engine *Engine) Run(addr string) {
	engine.Init()
	if err := http.ListenAndServe(addr, engine); err != nil {
		panic(err)
	}
}
