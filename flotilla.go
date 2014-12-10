package flotilla

import (
	"fmt"
	"net/http"

	"github.com/thrisp/engine"
)

type (
	// The base of running a Flotilla instance is an App struct with a Name,
	// an Env with information specific to running the App, and a chain of
	// Blueprints
	App struct {
		engine        *engine.Engine
		name          string
		Configured    bool
		Configuration []Configuration
		*Env
		*Blueprint
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
	app.Blueprint = RegisteredBlueprint("/", app)
	app.name = name
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

func (app *App) Name() string {
	return app.name
}

// ServeHTTP implements the http.Handler interface for the App.
func (app *App) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	app.engine.ServeHTTP(w, req)
}

func (app *App) Run(addr string) {
	if !app.Configured {
		if err := app.Configure(app.Configuration...); err != nil {
			panic(fmt.Sprintf("[FLOTILLA] app could not be configured properly: %s", err))
		}
	}
	if err := http.ListenAndServe(addr, app); err != nil {
		panic(err)
	}
}
