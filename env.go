package fleet

import (
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
	f := &FleetEnv{}
	f.configureFromFile(flpth)
	return f
}

func NewMapEnv(m map[string]string) *FleetEnv {
	f := &FleetEnv{}
	f.configureFromMap(m)
	return f
}

func (f *FleetEnv) configureFromMap(conf FleetConf) {
	f.FleetConf = conf
}

func (f *FleetEnv) configureFromFile(confpath string) {
	raw, _ := f.LoadConfFile(confpath)
	if raw != nil {
		f.FleetConf = raw
	}
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
