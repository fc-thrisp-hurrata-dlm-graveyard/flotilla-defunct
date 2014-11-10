package flotilla

import (
	"strings"

	"github.com/thrisp/engine"
)

var (
	configurelast []Configuration = []Configuration{ctemplating,
		csession,
		cengine}
)

type (
	// A function that takes an App pointer to configure the App.
	Configuration func(*App) error
)

// Configure takes any number of Configuration functions and to run the app through.
func (a *App) Configure(c ...Configuration) error {
	var err error
	c = append(c, configurelast...)
	for _, fn := range c {
		err = fn(a)
	}
	if err != nil {
		return err
	}
	a.Configured = true
	return nil
}

func cengine(a *App) error {
	e := a.engine
	if mm, err := a.Env.Store["UPLOAD_SIZE"].Int64(); err == nil {
		e.SetConf(engine.MaxFormMemory(mm))
	}
	if a.Mode == prodmode {
		e.SetConf(engine.ServePanic(false))
	}
	return nil
}

func csession(a *App) error {
	a.Env.SessionInit()
	return nil
}

func ctemplating(a *App) error {
	a.Env.TemplatorInit()
	return nil
}

// Mode takes a string for development, production, or testing to set the App mode.
func Mode(mode string) Configuration {
	return func(a *App) error {
		if existsIn(mode, []string{"development", "testing", "production"}) {
			a.SetMode(mode)
			return nil
		} else {
			return newError("mode must be development, testing, production, received %s", mode)
		}
	}
}

// EnvItem adds strings of the form "section_label:value" or "label:value" to
// the Env store, bypassing and without reading a conf file.
func EnvItem(items ...string) Configuration {
	return func(a *App) error {
		for _, item := range items {
			v := strings.Split(item, ":")
			k, value := v[0], v[1]
			sl := strings.Split(k, "_")
			if len(sl) > 1 {
				section, label := sl[0], sl[1]
				a.Env.Store.add(section, label, value)
			} else {
				a.Env.Store.add("", sl[0], value)
			}
		}
		return nil
	}
}

// CtxFunc adds a single function accessible as a Context Function.
func CtxFunc(name string, fn interface{}) Configuration {
	return func(a *App) error {
		return a.Env.AddCtxFunc(name, fn)
	}
}

// CtxFuncs adds functions accessible as Context Function.
func CtxFuncs(fns envmap) Configuration {
	return func(a *App) error {
		return a.Env.AddCtxFuncs(fns)
	}
}

// Templating supplies a Templator to the App.
func Templating(t Templator) Configuration {
	return func(a *App) error {
		a.Env.Templator = t
		return nil
	}
}

func TemplateFunction(name string, fn interface{}) Configuration {
	return func(a *App) error {
		a.Env.AddTplFunc(name, fn)
		return nil
	}
}

func TemplateFunctions(fns map[string]interface{}) Configuration {
	return func(a *App) error {
		a.Env.AddTplFuncs(fns)
		return nil
	}
}

// TemplateFunc adds a single context processor to the App primary RouteGroup.
func CtxProcessor(name string, fn interface{}) Configuration {
	return func(a *App) error {
		a.CtxProcessor(name, fn)
		return nil
	}
}

// TemplateFuncs adds a map of context processors to the App primary RouteGroup.
func CtxProcessors(fns ctxmap) Configuration {
	return func(a *App) error {
		a.CtxProcessors(fns)
		return nil
	}
}
