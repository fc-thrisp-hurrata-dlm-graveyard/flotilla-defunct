package flotilla

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thrisp/flotilla/session"
)

const (
	devmode = iota
	prodmode
	testmode
)

var (
	FlotillaPath     string
	workingPath      string
	workingStatic    string
	workingTemplates string
	defaultmode      int = devmode
)

type (
	// A function that takes an App pointer to configure the App.
	Configuration func(*App) error

	envmap map[string]interface{}

	// The App environment containing configuration variables & their store
	// as well as other information/data structures relevant to the app.
	Env struct {
		Mode int
		Store
		SessionManager *session.Manager
		Assets
		Templator
		flotilla     map[string]Flotilla
		ctxfunctions envmap
		tplfunctions envmap
	}
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

func (e *Env) defaults() {
	e.Store.adddefault("upload", "size", "10000000") // bytes
	e.Store.adddefault("secret", "key", "-")         // weak default value
}

// BaseEnv produces a base environment useful to new app instances.
func BaseEnv() *Env {
	e := &Env{Store: make(Store)}
	e.Templator = NewTemplator(e)
	e.ctxfunctions = make(envmap)
	e.AddCtxFuncs(builtinctxfuncs)
	e.tplfunctions = make(envmap)
	e.AddTplFuncs(builtintplfuncs)
	e.defaults()
	return e
}

// Merges an outside env instance with the existing app.Env
func (env *Env) MergeEnv(me *Env) {
	env.MergeStore(me.Store)
	for _, fs := range me.Assets {
		env.Assets = append(env.Assets, fs)
	}
	for _, dir := range me.StaticDirs() {
		env.AddStaticDir(dir)
	}
	env.AddTemplatesDir(me.Templator.ListTemplateDirs()...)
	env.AddCtxFuncs(me.ctxfunctions)
}

// MergeStore merges a Store instance with the Env's Store, without replacement.
func (env *Env) MergeStore(s Store) {
	for k, v := range s {
		if _, ok := env.Store[k]; !ok {
			env.Store[k] = v
		}
	}
}

// MergeFlotilla adds Flotilla to the Env
func (env *Env) MergeFlotilla(name string, f Flotilla) {
	if env.flotilla == nil {
		env.flotilla = make(map[string]Flotilla)
	}
	env.flotilla[name] = f
}

// Mode takes a string for development, production, or testing to set the App mode.
// Used in flotilla.New() or *App.SetConf()
func Mode(mode string) Configuration {
	return func(a *App) error {
		a.SetMode(mode)
		return nil
	}
}

// SetMode sets the running mode for the App env by a string.
func (env *Env) SetMode(value string) {
	switch value {
	case "development":
		env.Mode = devmode
	case "production":
		env.Mode = prodmode
	case "testing":
		env.Mode = testmode
	default:
		env.Mode = defaultmode
	}
}

// A string array of static dirs set in env.Store["staticdirectories"]
func (env *Env) StaticDirs() []string {
	if static, ok := env.Store["STATIC_DIRECTORIES"]; ok {
		if ret, err := static.List(); err == nil {
			return ret
		}
	}
	return []string{}
}

// AddStaticDir adds a static directory to be searched when a static route is accessed.
func (env *Env) AddStaticDir(dirs ...string) {
	if _, ok := env.Store["STATIC_DIRECTORIES"]; !ok {
		env.Store.add("static", "directories", "")
	}
	env.Store["STATIC_DIRECTORIES"].updateList(dirs...)
}

// TemplateDirs produces a listing of templator template directories.
func (env *Env) TemplateDirs() []string {
	return env.Templator.ListTemplateDirs()
}

// AddTemplatesDir adds a templates directory to the templator
func (env *Env) AddTemplatesDir(dirs ...string) {
	env.Templator.UpdateTemplateDirs(dirs...)
}

// AddTplFuncs adds template functions used by the default Templator.
func (env *Env) AddTplFuncs(fns envmap) {
	for k, v := range fns {
		env.tplfunctions[k] = v
	}
}

// AddCtxFuncs stores cross-handler functions in the Env as intermediate staging
// for later use by R context.
func (env *Env) AddCtxFuncs(fns envmap) {
	for k, v := range fns {
		env.ctxfunctions[k] = v
	}
}

func (env *Env) defaultsessionconfig() string {
	secret := env.Store["SECRET_KEY"].value
	return fmt.Sprintf(`{"cookieName":"flotillasessionid","enableSetCookie":false,"gclifetime":3600,"ProviderConfig":"{\"maxage\": 9000,\"cookieName\":\"flotillasessionid\",\"securityKey\":\"%s\"}"}`, secret)
}

func (env *Env) defaultsessionmanager() (*session.Manager, error) {
	return session.NewManager("cookie", env.defaultsessionconfig())
}

// SessionInit initializes the session using the SessionManager, or default if
// no session manage is specified.
func (env *Env) SessionInit() {
	if env.SessionManager == nil {
		sm, err := env.defaultsessionmanager()
		if err == nil {
			env.SessionManager = sm
		} else {
			panic(fmt.Sprintf("Problem with default session manager: %s", err))
		}
	}
	go env.SessionManager.GC()
}

func init() {
	workingPath, _ = os.Getwd()
	workingPath, _ = filepath.Abs(workingPath)
	workingStatic, _ = filepath.Abs("./static")
	workingTemplates, _ = filepath.Abs("./templates")
	FlotillaPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
}
