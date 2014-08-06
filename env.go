package fleet

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/thrisp/triago"
	//"./triago"
)

//const (
//	development = iota
//	testing
//	production
//)

var (
	defaultEnv = `HttpPort = 8080
AppName = fleet
RunInMode = development
StaticBase = static
`
	FleetPath     string
	workingPath   string
	workingStatic string
	err           error
)

type (
	FleetEnv struct {
		*triago.Config
		ConfPaths   []string
		StaticPaths []string
	}
)

func getDefaultEnv() []byte {
	env := fmt.Sprintf(defaultEnv, workingStatic)
	return []byte(env)
}

func newFleetConf(conf *triago.Config) *triago.Config {
	conf.EnvPrefix = "FLEET_"
	conf.FileName = filepath.Join(workingPath, "fleet.conf")
	return conf
}

func NewFleetEnv(filepaths ...string) *FleetEnv {
	f := &FleetEnv{Config: newFleetConf(triago.NewDefault())}
	f.makeConfFile(f.Config.FileName)
	f.AddConfPath(f.Config.FileName)
	for _, fp := range filepaths {
		if fp != "" {
			f.AddConfPath(fp)
		}
	}
	f.initEnv()
	return f
}

func (f *FleetEnv) makeConfFile(filePath string) {
	dirPath, _ := filepath.Split(filePath)

	if _, err = os.Stat(filePath); err != nil {
		if !os.IsNotExist(err) {
			return
		}
		if err = os.MkdirAll(dirPath, 0755); err != nil {
			return
		}
		if err = ioutil.WriteFile(filePath, getDefaultEnv(), 0644); err != nil {
			return
		}
	}
}

func (f *FleetEnv) AddConfPath(path string) {
	f.ConfPaths = append(f.ConfPaths, path)
}

func (f *FleetEnv) AddConfs(paths ...string) {
	for _, filePath := range paths {
		f.AddConfPath(filePath)
	}
	f.compileConf()
}

func (f *FleetEnv) mergeConf(path string) {
	m, err := triago.ReadDefault(path)
	if err != nil {
		fmt.Printf("%+v\n", err) //log
	}
	f.Merge(m)
}

func (f *FleetEnv) compileConf() {
	for _, filePath := range f.ConfPaths {
		f.mergeConf(filePath)
	}

	f.WriteFile(f.FileName,
		0777,
		fmt.Sprintf("fleet init conf :: %s", time.Now()))

	newconf, err := triago.ReadDefault(f.FileName)

	if err == nil {
		f.Config = newFleetConf(newconf)
	}
}

func (f *FleetEnv) initEnv() {
	// log
	f.compileConf()
}

func (f *FleetEnv) AddStaticPath(staticpath string) {
	if filepath.IsAbs(staticpath) {
		f.StaticPaths = append(f.StaticPaths, staticpath)
	} else {
		f.StaticPaths = append(f.StaticPaths, filepath.Join(workingPath, staticpath))
	}
}

func (f *FleetEnv) EnvQuery(section string, value string) (ret string) {
	ret, err := f.String(section, value)
	if err != nil {
		return fmt.Sprintf("%+v", err)
	}
	return ret
}

func init() {
	workingPath, _ = os.Getwd()
	workingPath, _ = filepath.Abs(workingPath)
	workingStatic, _ = filepath.Abs("./static")
	FleetPath, _ = filepath.Abs(filepath.Dir(os.Args[0]))
}
