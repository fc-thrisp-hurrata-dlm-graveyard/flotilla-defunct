package flotilla

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"reflect"

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
		staticdirectories []string
		ctxfunctions      map[string]interface{}
		Mode              int
		Conf
		Assets
		Templator
	}
)

func BaseEnv() *Env {
	e := &Env{Conf: make(map[string]string)}
	e.Templator = NewTemplator(e)
	e.AddCtxFuncs(builtinctxfuncs)
	return e
}

// Merges an env instance with the calling env
func (env *Env) MergeEnv(mergeenv *Env) {
	env.LoadConfMap(mergeenv.Conf)
	for _, fs := range mergeenv.Assets {
		env.Assets = append(env.Assets, fs)
	}
	for _, dir := range mergeenv.StaticDirs() {
		env.AddStaticDir(dir)
	}
	env.AddTemplatesDir(mergeenv.Templator.ListTemplateDirs()...)
	env.AddCtxFuncs(mergeenv.ctxfunctions)
}

// Loads a conf file into the env from the engine
func (engine *Engine) EnvConfFile(flpth string) bool {
	err := engine.Env.LoadConfFile(flpth)
	if err == nil {
		return true
	}
	return false
}

// Loads a conf map into the env from engine
func (engine *Engine) EnvConfMap(m map[string]string) bool {
	err := engine.Env.LoadConfMap(m)
	if err == nil {
		return true
	}
	return false
}

// Loads a conf file into the env
func (env *Env) LoadConfFile(filename string) (err error) {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	err = env.Conf.parse(reader, filename)
	return err
}

// Loads a conf as byte into the env
func (env *Env) LoadConfByte(b []byte, name string) (err error) {
	reader := bufio.NewReader(bytes.NewReader(b))
	err = env.Conf.parse(reader, name)
	return err
}

// Loads a conf as map into the env
func (env *Env) LoadConfMap(m map[string]string) (err error) {
	err = env.Conf.parsemap(m)
	return err
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

// All static directories specified for the engine including those set directly
// with the engine and those set in a configuration file
func (env *Env) StaticDirs() []string {
	dirs := env.staticdirectories
	if c, err := env.List("staticdirectories"); err == nil {
		for _, d := range c {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

// Adds a static directory to be searched when a static route is accessed.
func (env *Env) AddStaticDir(dir string) {
	env.staticdirectories = dirAdd(dir, env.staticdirectories)
}

func (env *Env) TemplateDirs() []string {
	return env.Templator.ListTemplateDirs()
}

// Adds a templates directory to the templator
func (env *Env) AddTemplatesDir(dirs ...string) {
	env.Templator.UpdateTemplateDirs(dirs...)
}

// Adds cross-handler functions from a map
func (env *Env) AddCtxFuncs(fns map[string]interface{}) {
	for k, v := range fns {
		env.AddCtxFunc(k, v)
	}
}

// Adds a cross-handler function by name/interface
func (env *Env) AddCtxFunc(name string, fn interface{}) {
	if env.ctxfunctions == nil {
		env.ctxfunctions = make(map[string]interface{})
	}

	env.ctxfunctions[name] = fn
}

// All env ctxfunctions available as reflect.Value(for use by *Ctx)
func (env *Env) CtxFunctions() map[string]reflect.Value {
	m := make(map[string]reflect.Value)
	for k, v := range env.ctxfunctions {
		m[k] = valueFunc(v)
	}
	return m
}

func (env *Env) parseFlags() {
	flagMode := flag.Flag("mode", "Run Flotilla app in mode: development, production or testing").Short('m').Default("development").String()
	flag.Parse()
	env.SetMode(*flagMode)
}

func init() {
	workingPath, _ = os.Getwd()
	workingPath, _ = filepath.Abs(workingPath)
	workingStatic, _ = filepath.Abs("./static")
	workingTemplates, _ = filepath.Abs("./templates")
	FlotillaPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
}
