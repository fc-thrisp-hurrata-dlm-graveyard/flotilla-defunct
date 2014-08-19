package fleet

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
)

var (
	FleetPath     string
	workingPath   string
	workingStatic string
)

type (
	FleetEnv struct {
		FleetConf
		ConfPath    string
		StaticPaths []string
	}
)

func NewFileEnv(flpth string) *FleetEnv {
	f := &FleetEnv{FleetConf: make(map[string]string)}
	f.LoadConfFile(flpth)
	return f
}

func NewMapEnv(m map[string]string) *FleetEnv {
	return &FleetEnv{FleetConf: m}
}

func (env *FleetEnv) LoadConfFile(filename string) (err error) {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	err = env.FleetConf.parse(reader, filename)
	return err
}

func (env *FleetEnv) LoadConfByte(b []byte, name string) (err error) {
	reader := bufio.NewReader(bytes.NewReader(b))
	err = env.FleetConf.parse(reader, name)
	return err
}

func (f *FleetEnv) AddStaticPath(staticpath string) {
	if filepath.IsAbs(staticpath) {
		f.StaticPaths = append(f.StaticPaths, staticpath)
	} else {
		f.StaticPaths = append(f.StaticPaths, filepath.Join(workingPath, staticpath))
	}
}

func init() {
	workingPath, _ = os.Getwd()
	workingPath, _ = filepath.Abs(workingPath)
	workingStatic, _ = filepath.Abs("./static")
	FleetPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
}
