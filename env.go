package flotilla

import (
	"fmt"
	"os"
	"path/filepath"

	"lcl/flotilla/session"

	flag "gopkg.in/alecthomas/kingpin.v1"
)

const (
	devmode  = iota
	prodmode = iota
	testmode = iota
)

var (
	FlotillaPath     string
	workingPath      string
	workingStatic    string
	workingTemplates string
	defaultmode      int = devmode
)

type (
	// The engine environment
	Env struct {
		Mode int
		Store
		SessionManager *session.Manager
		Assets
		Templator
		flotilla     map[string]Flotilla
		ctxfunctions map[string]interface{}
	}
)

func BaseEnv() *Env {
	e := &Env{Store: make(Store)}
	e.Templator = NewTemplator(e)
	e.ctxfunctions = make(map[string]interface{})
	e.AddCtxFuncs(builtinctxfuncs)
	e.Store.adddefault("secret", "key", "defaultsecretkeypleasechangethis")
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

// Merges an Store instance with the Env's Store
func (env *Env) MergeStore(s Store) {
	for k, v := range s {
		if _, ok := env.Store[k]; !ok {
			env.Store[k] = v
		}
	}
}

// Sets the running mode for the engine env by a string
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
	if static, ok := env.Store["staticdirectories"]; ok {
		if ret, err := static.getList(); err == nil {
			return ret
		}
	}
	return []string{}
}

// Adds a static directory to be searched when a static route is accessed.
func (env *Env) AddStaticDir(dirs ...string) {
	if _, ok := env.Store["staticdirectories"]; !ok {
		env.Store.add("", "staticdirectories", "")
	}
	env.Store["staticdirectories"].updateList(dirs...)
}

// Listing of templator template directories
func (env *Env) TemplateDirs() []string {
	return env.Templator.ListTemplateDirs()
}

// Adds a templates directory to the templator
func (env *Env) AddTemplatesDir(dirs ...string) {
	env.Templator.UpdateTemplateDirs(dirs...)
}

// Adds cross-handler functions
func (env *Env) AddCtxFuncs(fns map[string]interface{}) {
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
	secret := env.Store["secret_key"].value
	return fmt.Sprintf(`{"cookieName":"flotillasessionid","enableSetCookie":false,"gclifetime":3600,"ProviderConfig":"{\"cookieName\":\"flotillasessionid\",\"securityKey\":\"%s\"}"}`, secret)
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
	go env.SessionManager.GC()
}

func init() {
	workingPath, _ = os.Getwd()
	workingPath, _ = filepath.Abs(workingPath)
	workingStatic, _ = filepath.Abs("./static")
	workingTemplates, _ = filepath.Abs("./templates")
	FlotillaPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
}
