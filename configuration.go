package flotilla

import (
	"log"
	"os"
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
	var cnf []engine.Conf
	if mm, err := a.Env.Store["UPLOAD_SIZE"].Int64(); err == nil {
		cnf = append(cnf, engine.MaxFormMemory(mm))
	}
	if a.Mode.Production {
		cnf = append(cnf, engine.ServePanic(false))
	}
	if !a.Mode.Production {
		cnf = append(cnf, engine.Logger(log.New(os.Stdout, "[FLOTILLA]", 0)))
	}
	if err := e.SetConf(cnf...); err != nil {
		return err
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
func Mode(mode string, value bool) Configuration {
	return func(a *App) error {
		m := strings.Title(mode)
		if existsIn(m, []string{"Development", "Testing", "Production"}) {
			err := a.SetMode(m, value)
			if err != nil {
				return err
			}
		} else {
			return newError("mode must be Development, Testing, or Production; received %s", mode)
		}
		return nil
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

// CtxFuncs adds a map of functions accessible as Context Functions.
func CtxFuncs(fns map[string]interface{}) Configuration {
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

// TemplateFunction passes a template function to the env for Templator or
// other templating function use.
func TemplateFunction(name string, fn interface{}) Configuration {
	return func(a *App) error {
		a.Env.AddTplFunc(name, fn)
		return nil
	}
}

// TemplateFunction passes a map of functions to the env for Templator or
// other templating function use.
func TemplateFunctions(fns map[string]interface{}) Configuration {
	return func(a *App) error {
		a.Env.AddTplFuncs(fns)
		return nil
	}
}

// CtxProcessor adds a single template context processor to the App primary
// RouteGroup. This will affect all Routegroups & Routes.
func CtxProcessor(name string, fn interface{}) Configuration {
	return func(a *App) error {
		a.CtxProcessor(name, fn)
		return nil
	}
}

// CtxProcessors adds a map of context processors to the App primary RouteGroup.
func CtxProcessors(fns map[string]interface{}) Configuration {
	return func(a *App) error {
		a.CtxProcessors(fns)
		return nil
	}
}
