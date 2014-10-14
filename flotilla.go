package flotilla

import (
	"fmt"
	"net/http"

	"github.com/thrisp/engine"
)

type (
	// A HandlerFunc is any function taking a single parameter, *R
	HandlerFunc func(*R)

	// The base of running a Flotilla instance is an Engine struct with a Name,
	// an Env with information specific to running the engine, and a chain of
	// RouteGroups
	App struct {
		engine *engine.Engine
		Name   string
		*Env
		*RouteGroup
	}

	// A Blueprint struct is essential information about an engine for export
	// to another engine
	Blueprint struct {
		Name   string
		Prefix string
		Groups []*RouteGroup
		Env    *Env
	}

	// The Flotilla interface returns a Blueprint struct.
	Flotilla interface {
		Blueprint() *Blueprint
	}
)

// Returns an empty App instance
func Empty() *App {
	return &App{}
}

// Returns a new engine, with the minimum configuration.
func New(name string) *App {
	app := Empty()
	app.Env = BaseEnv()
	app.engine = app.defaultEngine()
	app.RouteGroup = NewRouteGroup("/", app)
	return app
}

// Returns a new engine instance with sensible defaults
func Basic() *App {
	app := New("flotilla")
	app.Use(Logger())
	app.STATIC("static")
	return app
}

func (a *App) defaultEngine() *engine.Engine {
	e, err := engine.New(engine.HTMLStatus(true))
	if err != nil {
		panic(fmt.Sprintf("Engine could not be created properly: %s", err))
	}
	return e
}

// Extend takes anytthing satisfying the Flotilla interface, and integrates it
// with the current Engine
func (app *App) Extend(f Flotilla) {
	blueprint := f.Blueprint()
	app.MergeFlotilla(blueprint.Name, f)
	app.MergeRouteGroups(blueprint.Groups)
	app.MergeEnv(blueprint.Env)
}

// Blueprint ensures the engine satisfies interface Flotilla by providing
// essential information in the engine in a struct: Name, RouteGroups, and Env
func (app *App) Blueprint() *Blueprint {
	return &Blueprint{Name: app.Name,
		Groups: app.Groups(),
		Env:    app.Env}
}

// Groups provides a flat array of RouteGroup instances attached to the App.
func (app *App) Groups() (groups RouteGroups) {
	type IterC func(r RouteGroups, fn IterC)

	groups = append(groups, app.RouteGroup)

	iter := func(r RouteGroups, fn IterC) {
		for _, x := range r {
			groups = append(groups, x)
			fn(x.children, fn)
		}
	}

	iter(app.children, iter)

	return groups
}

// Routes returns an array of Route instances, with all engine routes from all
// engine routergroups.
func (app *App) Routes() Routes {
	allroutes := make(Routes)
	for _, group := range app.Groups() {
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

// MergeRouteGroups merges an array of RouteGroup instances into the engine.
func (app *App) MergeRouteGroups(groups RouteGroups) {
	for _, x := range groups {
		if group, ok := app.existingGroup(x.prefix); ok {
			group.Use(x.Handlers...)
			app.MergeRoutes(group, x.routes)
		} else {
			newgroup := app.RouteGroup.New(x.prefix, x.Handlers...)
			app.MergeRoutes(newgroup, x.routes)
		}
	}
}

func (app *App) existingGroup(prefix string) (*RouteGroup, bool) {
	for _, g := range app.Groups() {
		if g.prefix == prefix {
			return g, true
		}
	}
	return nil, false
}

func (app *App) existingRoute(route *Route) bool {
	for _, r := range app.Routes() {
		if route.path == r.path {
			return true
		}
	}
	return false
}

// MergeRoutes merges the given group with the given routes, by route existence.
func (app *App) MergeRoutes(group *RouteGroup, routes Routes) {
	for _, route := range routes {
		if route.static && !app.existingRoute(route) {
			group.STATIC(route.path)
		}
		if !route.static && !app.existingRoute(route) {
			group.Handle(route)
		}
	}
}

func (app *App) init() {
	app.Env.SessionInit()
	// Send Flotilla configured items back down to the engine after all
	// configuration (should have) has taken place.
	if mm, err := app.Env.Store["UPLOAD_SIZE"].Int64(); err == nil {
		app.engine.SetConf(engine.MaxFormMemory(mm))
	}
	// engine panic on by mode
}

// ServeHTTP implements the http.Handler interface for the App.
func (app *App) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	app.engine.ServeHTTP(w, req)
}

func (app *App) Run(addr string) {
	app.init()
	if err := http.ListenAndServe(addr, app); err != nil {
		panic(err)
	}
}
