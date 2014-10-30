package flotilla

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/thrisp/engine"
)

type (
	// A HandlerFunc is any function taking a single parameter, *R
	HandlerFunc func(*Ctx)

	// The base of running a Flotilla instance is an App struct with a Name,
	// an Env with information specific to running the App, and a chain of
	// RouteGroups
	App struct {
		engine *engine.Engine
		Name   string
		*Env
		*RouteGroup
	}

	// A Transport struct contains essential information about an App for export
	// to another App.
	Transport struct {
		Name   string
		Prefix string
		Groups []*RouteGroup
		Env    *Env
	}

	// The Flotilla interface returns a Transport struct.
	Flotilla interface {
		Transport() *Transport
	}
)

// Returns an empty App instance with no configuration.
func Empty() *App {
	return &App{Env: EmptyEnv()}
}

// Returns a new App, with minimum configuration.
func New(name string, conf ...Configuration) *App {
	app := Empty()
	app.Env.NewEnv()
	err := app.SetConf(conf...)
	app.engine = app.defaultEngine()
	app.RouteGroup = NewRouteGroup("/", app)
	app.Name = name
	app.STATIC("static")
	if err != nil {
		panic(fmt.Sprintf("[FLOTILLA] problem creating new *App: %s", err))
	}
	return app
}

func (a *App) defaultEngine() *engine.Engine {
	e, err := engine.New(engine.HTMLStatus(true))
	if a.Mode != prodmode {
		e.SetConf(engine.Logger(log.New(os.Stdout, "[FLOTILLA]", 0)))
	}
	if err != nil {
		panic(fmt.Sprintf("[FLOTILLA] engine could not be created properly: %s", err))
	}
	return e
}

// Extend takes anything satisfying the Flotilla interface, and integrates it
// with the current Engine.
func (app *App) Extend(f Flotilla) {
	transport := f.Transport()
	app.MergeFlotilla(transport.Name, f)
	app.MergeRouteGroups(transport.Groups)
	app.MergeEnv(transport.Env)
}

// Transport ensures the App satisfies interface Flotilla by providing
// essential information in a struct: Name, RouteGroups, and Env.
func (app *App) Transport() *Transport {
	return &Transport{Name: app.Name,
		Groups: app.Groups(),
		Env:    app.Env}
}

// Groups provides a flat array of RouteGroup instances attached to the App.
func (app *App) Groups() (groups []*RouteGroup) {
	type IterC func(r []*RouteGroup, fn IterC)

	groups = append(groups, app.RouteGroup)

	iter := func(r []*RouteGroup, fn IterC) {
		for _, x := range r {
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
	if app.Mode == prodmode {
		app.engine.SetConf(engine.ServePanic(false))
	}
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
