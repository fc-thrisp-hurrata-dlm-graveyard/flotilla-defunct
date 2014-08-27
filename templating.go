package flotilla

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thrisp/jingo"
)

type (
	// An interface attached to an env as a templating base
	Templator interface {
		Render(io.Writer, string, interface{}) error
		ListTemplateDirs() []string
		UpdateTemplateDirs(...string)
	}

	// The default Flotilla templator
	templator struct {
		*jingo.Jingo
		TemplateDirs []string
	}

	// The default templator loader
	Loader struct {
		env            *Env
		FileExtensions []string
	}
)

func NewTemplator(e *Env) *templator {
	j := &templator{Jingo: jingo.NewJingo()}
	j.UpdateTemplateDirs(workingTemplates)
	j.AddLoaders(NewLoader(e))
	return j
}

func (t *templator) ListTemplateDirs() []string {
	return t.TemplateDirs
}

func (t *templator) UpdateTemplateDirs(dirs ...string) {
	for _, dir := range dirs {
		t.TemplateDirs = dirAdd(dir, t.TemplateDirs)
	}
}

func NewLoader(e *Env) *Loader {
	fl := &Loader{env: e}
	fl.FileExtensions = append(fl.FileExtensions, ".html", ".jingo")
	return fl
}

func (fl *Loader) ValidExtension(ext string) bool {
	for _, extension := range fl.FileExtensions {
		if extension == ext {
			return true
		}
	}
	return false
}

func (fl *Loader) ListTemplates() interface{} {
	return "flotilla loader ListTemplates not yet implemented"
}

func (fl *Loader) Load(name string) (string, error) {
	for _, p := range fl.env.TemplateDirs() {
		f := filepath.Join(p, name)
		if fl.ValidExtension(filepath.Ext(f)) {
			// existing template dirs
			if _, err := os.Stat(f); err == nil {
				file, err := os.Open(f)
				r, err := ioutil.ReadAll(file)
				return string(r), err
			}
			// binary assets
			if r, err := fl.env.Assets.Get(name); err == nil {
				r, err := ioutil.ReadAll(r)
				return string(r), err
			}
		}
	}
	return "", newError("Template %s does not exist", name)
}
