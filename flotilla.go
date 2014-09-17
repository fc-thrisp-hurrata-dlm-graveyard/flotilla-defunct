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
		HttpExceptions
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
	engine.RouterGroup = NewRouterGroup("/", engine)
	engine.router.NotFound = engine.handler404
	engine.router.PanicHandler = engine.handler500
	engine.cache.New = engine.newCtx
	engine.HttpExceptions = defaulthttpexceptions()
	return engine
}

// Returns a basic engine instance with sensible defaults
func Basic() *Engine {
	engine := New("flotilla")
	engine.Use(Logger())
	engine.STATIC("static")
	return engine
}

// Extends an engine with Flotilla interface
func (engine *Engine) Extend(f Flotilla) {
	blueprint := f.Blueprint()
	engine.MergeFlotilla(blueprint.Name, f)
	engine.MergeRouterGroups(blueprint.Groups)
	engine.MergeEnv(blueprint.Env)
}

// Blueprint ensures the engine satisfies interface Flotilla by providing
// essential information in the engine: Name, RouterGroups, and Env
func (engine *Engine) Blueprint() *Blueprint {
	return &Blueprint{Name: engine.Name,
		Groups: engine.Groups(),
		Env:    engine.Env}
}

// Groups provides an array of RouterGroup instances attached to the engine.
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

// An array of Route instances, with all engine routes from all engine routergroups.
func (engine *Engine) Routes() map[string]*Route {
	allroutes := make(map[string]*Route)
	for _, group := range engine.Groups() {
		for _, route := range group.routes {
			if route.Name != "" {
				allroutes[route.Name] = route
			} else {
				allroutes[route.Named()] = route
			}
		}
	}
	return allroutes
}

// Merges an array of RouterGroup instances into the engine.
func (engine *Engine) MergeRouterGroups(groups []*RouterGroup) {
	for _, x := range groups {
		if group, ok := engine.existingGroup(x.prefix); ok {
			group.Use(x.Handlers...)
			engine.MergeRoutes(group, x.routes)
		} else {
			newgroup := engine.RouterGroup.New(x.prefix, x.Handlers...)
			engine.MergeRoutes(newgroup, x.routes)
		}
	}
}

func (engine *Engine) existingGroup(prefix string) (*RouterGroup, bool) {
	for _, g := range engine.Groups() {
		if g.prefix == prefix {
			return g, true
		}
	}
	return nil, false
}

func (engine *Engine) existingRoute(route *Route) bool {
	for _, r := range engine.Routes() {
		if route.path == r.path {
			return true
		}
	}
	return false
}

// Merges the given group with the given routes based on route existence.
func (engine *Engine) MergeRoutes(group *RouterGroup, routes map[string]*Route) {
	for _, route := range routes {
		if route.static && !engine.existingRoute(route) {
			group.STATIC(route.path)
		}
		if !route.static && !engine.existingRoute(route) {
			group.Handle(route)
		}
	}
}

func (engine *Engine) init() {
	engine.parseFlags()
	engine.Env.SessionInit()
}

// ServeHTTP makes the router implement the http.Handler interface.
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	engine.router.ServeHTTP(w, req)
}

func (engine *Engine) Run(addr string) {
	engine.init()
	if err := http.ListenAndServe(addr, engine); err != nil {
		panic(err)
	}
}
