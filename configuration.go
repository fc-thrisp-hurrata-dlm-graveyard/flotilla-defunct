package flotilla

import "strings"

type (
	// A function that takes an App pointer to configure the App.
	Configuration func(*App) error
)

// SetConf takes any number of Configuration functions and to run the app through.
func (a *App) SetConf(configurations ...Configuration) error {
	for _, conf := range configurations {
		if err := conf(a); err != nil {
			return err
		}
	}
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
// the Env store, by bypassing and without reading a conf file.
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

// CtxFunc adds a single function accessible as a Ctx function.
func CtxFunc(name string, fn interface{}) Configuration {
	return func(a *App) error {
		return a.Env.AddCtxFunc(name, fn)
	}
}

// CtxFuncs adds functions accessible as Ctx function by map[string]interface{}
func CtxFuncs(fns envmap) Configuration {
	return func(a *App) error {
		return a.Env.AddCtxFuncs(fns)
	}
}

func Templating(t Templator) Configuration {
	return func(a *App) error {
		a.Env.Templator = t
		return nil
	}
}

// TemplateFunc adds a single function accessible within a template.
func TemplateFunc(name string, fn interface{}) Configuration {
	return func(a *App) error {
		return nil
	}
}

// TemplateFuncs adds functions accessible to templates.
func TemplateFuncs(fns envmap) Configuration {
	return func(a *App) error {
		return nil
	}
}
