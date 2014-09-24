package flotilla

import (
	"fmt"
	"os"
	"path/filepath"

	//github.com/thrisp/flotilla/session
	"lcl/flotilla/session"

	flag "gopkg.in/alecthomas/kingpin.v1"
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

	// The engine environment containing configuration variables & their store
	// as well as other information/data structures relevant to the engine.
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

func (e *Env) defaults() {
	e.Store.adddefault("secret", "key", "-")
}

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

// Merges an env instance with the calling env
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

// Merges a Store instance with the Env's Store, without replacement.
func (env *Env) MergeStore(s Store) {
	for k, v := range s {
		if _, ok := env.Store[k]; !ok {
			env.Store[k] = v
		}
	}
}

func (env *Env) MergeFlotilla(name string, f Flotilla) {
	if env.flotilla == nil {
		env.flotilla = make(map[string]Flotilla)
	}
	env.flotilla[name] = f
}

// Sets the running mode for the App env by a string.
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

// Adds a static directory to be searched when a static route is accessed.
func (env *Env) AddStaticDir(dirs ...string) {
	if _, ok := env.Store["STATIC_DIRECTORIES"]; !ok {
		env.Store.add("static", "directories", "")
	}
	env.Store["STATIC_DIRECTORIES"].updateList(dirs...)
}

// Listing of templator template directories
func (env *Env) TemplateDirs() []string {
	return env.Templator.ListTemplateDirs()
}

// Adds a templates directory to the templator
func (env *Env) AddTemplatesDir(dirs ...string) {
	env.Templator.UpdateTemplateDirs(dirs...)
}

// Adds template functions used by the default Templator.
func (env *Env) AddTplFuncs(fns envmap) {
	for k, v := range fns {
		env.tplfunctions[k] = v
	}
}

// Adds cross-handler functions used by the Ctx.
func (env *Env) AddCtxFuncs(fns envmap) {
	for k, v := range fns {
		env.ctxfunctions[k] = v
	}
}

func (env *Env) parseFlags() {
	flagMode := flag.Flag("mode", "Run Flotilla app in mode: development, production or testing").Short('m').Default("development").String()
	flag.Parse()
	env.SetMode(*flagMode)
}

func (env *Env) defaultsessionconfig() string {
	secret := env.Store["SECRET_KEY"].value
	return fmt.Sprintf(`{"cookieName":"flotillasessionid","enableSetCookie":false,"gclifetime":3600,"ProviderConfig":"{\"maxage\": 9000,\"cookieName\":\"flotillasessionid\",\"securityKey\":\"%s\"}"}`, secret)
}

func (env *Env) defaultsessionmanager() (*session.Manager, error) {
	return session.NewManager("cookie", env.defaultsessionconfig())
}

func (env *Env) SessionInit() {
	if env.SessionManager == nil {
		sm, err := env.defaultsessionmanager()
		if err == nil {
			env.SessionManager = sm
		} else {
			panic(fmt.Sprintf("Problem with default session manager: %s", err))
		}
	}
	fmt.Printf("%+v\n", env.SessionManager)
	go env.SessionManager.GC()
}

// Slice of flotilla interfaces of the current engine, starting with calling engine
//func (engine *Engine) Flotilla() []Flotilla {
//	var ret []Flotilla
//	ret = append(ret, engine)
//	for _, e := range engine.Env.flotilla {
//		ret = append(ret, e)
//	}
//	return ret
//}

func init() {
	workingPath, _ = os.Getwd()
	workingPath, _ = filepath.Abs(workingPath)
	workingStatic, _ = filepath.Abs("./static")
	workingTemplates, _ = filepath.Abs("./templates")
	FlotillaPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
}
