package flotilla

import (
	"lcl/engine"
	"net/http"
)

type (
	HandlerFunc func(*R)

	// The base of running a Flotilla instance is an Engine struct with a Name,
	// an Env with information specific to running the engine, and a chain of
	// RouterGroups
	App struct {
		engine *engine.Engine
		Name   string
		*Env
		*RouterGroup
	}

	// A Blueprint struct is essential information about an engine for export
	// to another engine
	Blueprint struct {
		Name   string
		Prefix string
		Groups []*RouterGroup
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
	app.engine = engine.New()
	app.Env = BaseEnv()
	app.RouterGroup = NewRouterGroup("/", app)
	//engine.router.NotFound = app.flotilla404
	//engine.router.PanicHandler = app.flotilla500
	return app
}

// Returns a new engine instance with sensible defaults
func Basic() *App {
	app := New("flotilla")
	app.Use(Logger())
	app.STATIC("static")
	return app
}

// Extend takes anytthing satisfying the Flotilla interface, and integrates it
// with the current Engine
func (app *App) Extend(f Flotilla) {
	blueprint := f.Blueprint()
	app.MergeFlotilla(blueprint.Name, f)
	app.MergeRouterGroups(blueprint.Groups)
	app.MergeEnv(blueprint.Env)
}

// Blueprint ensures the engine satisfies interface Flotilla by providing
// essential information in the engine in a struct: Name, RouterGroups, and Env
func (app *App) Blueprint() *Blueprint {
	return &Blueprint{Name: app.Name,
		Groups: app.Groups(),
		Env:    app.Env}
}

// Groups provides a flat array of RouterGroup instances attached to the App.
func (app *App) Groups() (groups RouterGroups) {
	type IterC func(r RouterGroups, fn IterC)

	groups = append(groups, app.RouterGroup)

	iter := func(r RouterGroups, fn IterC) {
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

// Merges an array of RouterGroup instances into the engine.
func (app *App) MergeRouterGroups(groups RouterGroups) {
	for _, x := range groups {
		if group, ok := app.existingGroup(x.prefix); ok {
			group.Use(x.Handlers...)
			app.MergeRoutes(group, x.routes)
		} else {
			newgroup := app.RouterGroup.New(x.prefix, x.Handlers...)
			app.MergeRoutes(newgroup, x.routes)
		}
	}
}

func (app *App) existingGroup(prefix string) (*RouterGroup, bool) {
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

// Merges the given group with the given routes based on route existence.
func (app *App) MergeRoutes(group *RouterGroup, routes Routes) {
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
	app.parseFlags()
	app.Env.SessionInit()
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
