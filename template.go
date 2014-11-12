package flotilla

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thrisp/djinn"
)

type (
	// Templator is an interface with methods for application templating.
	Templator interface {
		Render(io.Writer, string, interface{}) error
		ListTemplateDirs() []string
		ListTemplates() []string
		UpdateTemplateDirs(...string)
	}

	// The default Flotilla templator
	templator struct {
		*djinn.Djinn
		TemplateDirs []string
	}

	// The default templator loader
	Loader struct {
		env            *Env
		FileExtensions []string
	}
)

func NewTemplator(e *Env) *templator {
	j := &templator{Djinn: djinn.Empty()}
	j.UpdateTemplateDirs(workingTemplates)
	j.SetConf(djinn.Loaders(NewLoader(e)), djinn.TemplateFunctions(e.tplfunctions))
	return j
}

func (t *templator) ListTemplateDirs() []string {
	return t.TemplateDirs
}

func (t *templator) ListTemplates() []string {
	var ret []string
	for _, l := range t.Djinn.Loaders {
		ts := l.ListTemplates().([]string)
		ret = append(ret, ts...)
	}
	return ret
}

func (t *templator) UpdateTemplateDirs(dirs ...string) {
	for _, dir := range dirs {
		t.TemplateDirs = doAdd(dir, t.TemplateDirs)
	}
}

func NewLoader(e *Env) *Loader {
	fl := &Loader{env: e}
	fl.FileExtensions = append(fl.FileExtensions, ".html", ".dji")
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

// AssetTemplates returns a string array of templates in binary assets attached
// to the application. Iterates all assets, returns filenames matching flotilla
// loader valid extensions(default .html, .dji).
func (fl *Loader) AssetTemplates() []string {
	var ret []string
	for _, assetfs := range fl.env.Assets {
		for _, f := range assetfs.AssetNames() {
			if fl.ValidExtension(filepath.Ext(f)) {
				ret = append(ret, f)
			}
		}
	}
	return ret
}

// ListTemplates returns a string array of absolute template paths for all
// templates dirs & assets matching valid extensions(default .html, .dji) and
// associated with the flotilla loader.
func (fl *Loader) ListTemplates() interface{} {
	var ret []string
	for _, p := range fl.env.TemplateDirs() {
		files, _ := ioutil.ReadDir(p)
		for _, f := range files {
			if fl.ValidExtension(filepath.Ext(f.Name())) {
				ret = append(ret, fmt.Sprintf("%s/%s", p, f.Name()))
			}
		}
	}
	ret = append(ret, fl.AssetTemplates()...)
	return ret
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
