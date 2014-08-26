package flotilla

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
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
		Conf
		StaticDirs []string
		Mode       int
		Templator
		Assets
	}
)

func BaseEnv() *Env {
	e := &Env{Conf: make(map[string]string),
		Mode: defaultmode}
	e.Templator = NewTemplator(e)
	return e
}

// Merges an env instance with the calling env
func (env *Env) MergeEnv(mergeenv *Env) {
	env.LoadConfMap(mergeenv.Conf)
	for _, fs := range mergeenv.Assets {
		env.Assets = append(env.Assets, fs)
	}
	for _, dir := range mergeenv.StaticDirs {
		env.AddStaticDir(dir)
	}
	env.AddTemplatesDir(mergeenv.Templator.ListTemplateDirs()...)
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

// Loads a conf as []byte into the env
func (env *Env) LoadConfByte(b []byte, name string) (err error) {
	reader := bufio.NewReader(bytes.NewReader(b))
	err = env.Conf.parse(reader, name)
	return err
}

// Loads a conf as a map[string]string into the env
func (env *Env) LoadConfMap(m map[string]string) (err error) {
	err = env.Conf.parsemap(m)
	return err
}

// Sets the running mode for the engine env
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

// Adds a static directory to be searched when a static route is accessed.
func (env *Env) AddStaticDir(dir string) {
	env.StaticDirs = dirAdd(dir, env.StaticDirs)
}

// Adds a templates directory to the templator
func (env *Env) AddTemplatesDir(dir ...string) {
	env.Templator.UpdateTemplateDirs(dir...)
}

func init() {
	workingPath, _ = os.Getwd()
	workingPath, _ = filepath.Abs(workingPath)
	workingStatic, _ = filepath.Abs("./static")
	workingTemplates, _ = filepath.Abs("./templates")
	FlotillaPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
}
