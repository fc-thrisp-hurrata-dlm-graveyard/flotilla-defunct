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
	envmap map[string]interface{}

	// The App environment containing configuration variables & their store
	// as well as other info & data relevant to the app.
	Env struct {
		Mode int
		Store
		SessionManager *session.Manager
		Assets
		Templator
		flotilla     map[string]Flotilla
		ctxfunctions envmap
	}
)

func (e *Env) defaults() {
	e.Store.adddefault("upload", "size", "10000000")   // bytes
	e.Store.adddefault("secret", "key", "change-this") // weak default value
	e.Store.adddefault("session", "cookiename", "session")
	e.Store.adddefault("session", "lifetime", "2629743")
}

// EmptyEnv produces an Env with intialization but no configuration.
func EmptyEnv() *Env {
	e := &Env{Store: make(Store)}
	e.ctxfunctions = make(envmap)
	return e
}

// NewEnv configures an intialized Env.
func (env *Env) BaseEnv() {
	env.AddCtxFuncs(builtinctxfuncs)
	env.defaults()
}

// Merges an outside env instance with the calling Env.
func (env *Env) MergeEnv(o *Env) {
	env.MergeStore(o.Store)
	for _, fs := range o.Assets {
		env.Assets = append(env.Assets, fs)
	}
	for _, dir := range o.StaticDirs() {
		env.AddStaticDir(dir)
	}
	env.AddTemplatesDir(o.Templator.ListTemplateDirs()...)
	env.AddCtxFuncs(o.ctxfunctions)
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

// AddCtxFunc adds a single Ctx function with the name string, checking that
// the function is a valid function returning 1 value, or 1 value and 1 error
// value.
func (env *Env) AddCtxFunc(name string, fn interface{}) error {
	err := validctxfunc(fn)
	if err == nil {
		env.ctxfunctions[name] = fn
		return nil
	}
	return err
}

// AddCtxFuncs stores cross-handler functions in the Env as intermediate staging
// for later use by Ctx.
func (env *Env) AddCtxFuncs(fns envmap) error {
	for k, v := range fns {
		err := env.AddCtxFunc(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (env *Env) defaultsessionconfig() string {
	secret := env.Store["SECRET_KEY"].value
	cookie_name := env.Store["SESSION_COOKIENAME"].value
	session_lifetime, _ := env.Store["SESSION_LIFETIME"].Int64()
	prvdrcfg := fmt.Sprintf(`"ProviderConfig":"{\"maxage\": %d,\"cookieName\":\"%s\",\"securityKey\":\"%s\"}"`, session_lifetime, cookie_name, secret)
	return fmt.Sprintf(`{"cookieName":"%s","enableSetCookie":false,"gclifetime":3600, %s}`, cookie_name, prvdrcfg)
}

func (env *Env) defaultsessionmanager() *session.Manager {
	d, err := session.NewManager("cookie", env.defaultsessionconfig())
	if err != nil {
		panic(fmt.Sprintf("Problem with [FLOTILLA] default session manager: %s", err))
	}
	return d
}

// SessionInit initializes the session using the SessionManager, or default if
// no session manage is specified.
func (env *Env) SessionInit() {
	if env.SessionManager == nil {
		env.SessionManager = env.defaultsessionmanager()
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
