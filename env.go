package flotilla

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
)

var (
	FlotillaPath  string
	workingPath   string
	workingStatic string
)

type (
	Env struct {
		Conf
		StaticPaths []string
	}
)

func BaseEnv() *Env {
	return &Env{Conf: make(map[string]string)}
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

func (env *Env) AddStaticPath(staticpath string) {
	if filepath.IsAbs(staticpath) {
		env.StaticPaths = append(env.StaticPaths, staticpath)
	} else {
		env.StaticPaths = append(env.StaticPaths, filepath.Join(workingPath, staticpath))
	}
}

func init() {
	workingPath, _ = os.Getwd()
	workingPath, _ = filepath.Abs(workingPath)
	workingStatic, _ = filepath.Abs("./static")
	FlotillaPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
}
