package flotilla

import (
	"fmt"
	"net/http"
)

type (
	// The base of running a Flotilla instance is an App struct with a Name,
	// an Env with information specific to running the App, and a chain of
	// Blueprints
	App struct {
		name string
		Engine
		*Config
		*Env
		*Blueprint
	}
)

// Returns an empty App instance with no configuration.
func Empty(name string) *App {
	return &App{name: name, Env: EmptyEnv()}
}

// Returns a new App with the provided Engine and minimum configuration.
func New(name string, efn SetEngine, conf ...Configuration) *App {
	app := Empty(name)
	app.BaseEnv()
	app.Config = defaultConfig()
	efn(app)
	app.Blueprint = NewBlueprint("/")
	app.STATIC("static")
	app.Configured = false
	app.Configuration = conf
	return app
}

func (app *App) Name() string {
	return app.name
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
