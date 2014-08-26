package flotilla

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"

	"github.com/thrisp/jingo"
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
	Env struct {
		Conf
		StaticDirs []string
		Mode       int
		Templator  *jingo.Jingo
		Assets
	}
)

func BaseEnv() *Env {
	e := &Env{Conf: make(map[string]string),
		Mode: defaultmode}
	e.Templator = NewTemplator(e)
	return e
}

func (engine *Engine) NewFileEnv(flpth string) bool {
	err := engine.Env.LoadConfFile(flpth)
	if err == nil {
		return true
	}
	return false
}

func (engine *Engine) NewMapEnv(m map[string]string) bool {
	err := engine.Env.LoadConfMap(m)
	if err == nil {
		return true
	}
	return false
}

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

func (env *Env) LoadConfByte(b []byte, name string) (err error) {
	reader := bufio.NewReader(bytes.NewReader(b))
	err = env.Conf.parse(reader, name)
	return err
}

func (env *Env) LoadConfMap(m map[string]string) (err error) {
	err = env.Conf.parsemap(m)
	return err
}

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

func (env *Env) AddStaticDir(dir string) {
	env.StaticDirs = env.dirAdd(dir, env.StaticDirs)
}

// adds a dir(checked as absolute & appendable) to the first FlotillaLoader found in
// Env.Templator.(*jingo.Jingo)
func (env *Env) AddTemplatesDir(dir string) {
	for _, l := range env.Templator.Loaders {
		if _, ok := l.(*FlotillaLoader); ok {
			sd := l.(*FlotillaLoader).TemplateDirs
			l.(*FlotillaLoader).TemplateDirs = env.dirAdd(dir, sd)
			break
		}
	}
}

func (env *Env) dirAdd(dir string, envDirs []string) []string {
	adddir := env.dirAbs(dir)
	if env.dirAppendable(adddir, envDirs) {
		envDirs = append(envDirs, adddir)
	}
	return envDirs
}

func (env *Env) dirAbs(dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	} else {
		return filepath.Join(workingPath, dir)
	}
}

func (env *Env) dirAppendable(dir string, envDirs []string) bool {
	for _, d := range envDirs {
		if d == dir {
			return false
		}
	}
	return true
}

func init() {
	workingPath, _ = os.Getwd()
	workingPath, _ = filepath.Abs(workingPath)
	workingStatic, _ = filepath.Abs("./static")
	workingTemplates, _ = filepath.Abs("./templates")
	FlotillaPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
}
