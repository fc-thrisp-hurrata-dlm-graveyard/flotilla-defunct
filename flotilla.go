package flotilla

import (
	"fmt"
	"net/http"

	"github.com/thrisp/engine"
)

type (
	// A HandlerFunc is any function taking a single parameter, *Ctx
	HandlerFunc func(*Ctx)

	// The base of running a Flotilla instance is an App struct with a Name,
	// an Env with information specific to running the App, and a chain of
	// RouteGroups
	App struct {
		engine        *engine.Engine
		Name          string
		Configured    bool
		Configuration []Configuration
		*Env
		*RouteGroup
	}

	// A Blueprint struct contains essential information about an App for export
	// to another App.
	Blueprint struct {
		Name   string
		Groups []*RouteGroup
		Env    *Env
	}

	// The Flotilla interface returns a Blueprint struct.
	Flotilla interface {
		Blueprint() *Blueprint
	}
)

// Returns an empty App instance with no configuration.
func Empty() *App {
	return &App{Env: EmptyEnv()}
}

// Returns a new App, with minimum configuration.
func New(name string, conf ...Configuration) *App {
	app := Empty()
	app.BaseEnv()
	app.engine = app.defaultEngine()
	app.RouteGroup = NewRouteGroup("/", app)
	app.Name = name
	app.STATIC("static")
	app.Configured = false
	app.Configuration = conf
	return app
}

func (a *App) defaultEngine() *engine.Engine {
	e, err := engine.New(engine.HTMLStatus(true))
	if err != nil {
		panic(fmt.Sprintf("[FLOTILLA] engine could not be created properly: %s", err))
	}
	return e
}

// Extend takes anything satisfying the Flotilla interface, and integrates it
// with the current Engine.
func (app *App) Extend(f Flotilla) {
	blueprint := f.Blueprint()
	app.MergeFlotilla(blueprint.Name, f)
	app.MergeRouteGroups(blueprint.Groups)
	app.MergeEnv(blueprint.Env)
}

// Blueprint ensures the App satisfies interface Flotilla by providing
// essential information in a struct: Name, RouteGroups, and Env.
func (app *App) Blueprint() *Blueprint {
	return &Blueprint{Name: app.Name,
		Groups: app.Groups(),
		Env:    app.Env}
}

// Groups provides a flat array of RouteGroup instances attached to the App.
func (app *App) Groups() (groups []*RouteGroup) {
	type IterC func(rs []*RouteGroup, fn IterC)

	groups = append(groups, app.RouteGroup)

	iter := func(rs []*RouteGroup, fn IterC) {
		for _, x := range rs {
			groups = append(groups, x)
			fn(x.children, fn)
		}
	}

	iter(app.children, iter)

	return groups
}

// Routes returns an array of Route instances, with all App routes from all
// App routergroups.
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

// MergeRouteGroups merges an array of RouteGroup instances into the App.
func (app *App) MergeRouteGroups(groups []*RouteGroup) {
	for _, g := range groups {
		if group, ok := app.existingGroup(g.prefix); ok {
			group.Use(g.Handlers...)
			app.MergeRoutes(group, g.routes)
		} else {
			newgroup := app.RouteGroup.New(g.prefix, g.Handlers...)
			app.MergeRoutes(newgroup, g.routes)
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

// ServeHTTP implements the http.Handler interface for the App.
func (app *App) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	app.engine.ServeHTTP(w, req)
}

func (app *App) Run(addr string) {
	if !app.Configured {
		if err := app.Configure(app.Configuration...); err != nil {
			panic(err)
		}
	}
	if err := http.ListenAndServe(addr, app); err != nil {
		panic(err)
	}
}
