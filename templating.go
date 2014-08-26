package flotilla

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thrisp/jingo"
)

type (
	FlotillaLoader struct {
		jingo.DirLoader
		TemplateDirs []string
		env          *Env
	}
)

func NewTemplator(e *Env) *jingo.Jingo {
	j := jingo.NewJingo()
	fl := NewFlotillaLoader(e)
	j.AddLoaders(fl)
	return j
}

func NewFlotillaLoader(e *Env) *FlotillaLoader {
	fl := &FlotillaLoader{env: e}
	fl.FileExtensions = append(fl.FileExtensions, ".html", ".jingo")
	fl.addTemplateDir(workingTemplates)
	return fl
}

func (fl *FlotillaLoader) addTemplateDir(dir string) {
	fl.TemplateDirs = append(fl.TemplateDirs, dir)
}

func (fl *FlotillaLoader) ListTemplates() interface{} {
	return "flotilla loader list templates not implemented"
}

func (fl *FlotillaLoader) Load(name string) (string, error) {
	for _, p := range fl.TemplateDirs {
		f := filepath.Join(p, name)
		if fl.ValidExtension(filepath.Ext(f)) {
			if _, err := os.Stat(f); err == nil {
				file, err := os.Open(f)
				r, err := ioutil.ReadAll(file)
				return string(r), err
			}
			//assets
		}
	}
	return "", newError("Template %s does not exist", name)
}
