package flotilla

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/thrisp/djinn"
	"github.com/thrisp/flotilla/session"
)

type (
	// An interface attached to an app env to handle templating.
	Templator interface {
		Render(io.Writer, string, interface{}) error
		ListTemplateDirs() []string
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

	// TData is a struct sent to and accessible within the template, by the
	// builtin rendertemplate function.
	TData struct {
		Any     interface{}
		Request *http.Request
		Session session.SessionStore
		Data    ctxdata
		Flash   map[string]string
	}
)

func NewTemplator(e *Env) *templator {
	j := &templator{Djinn: djinn.Empty()}
	j.UpdateTemplateDirs(workingTemplates)
	j.SetConf(djinn.Loaders(NewLoader(e)), djinn.TemplateFunctions(builtintplfuncs))
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

func (t *templator) UpdateTemplateFuncs(funcs map[string]interface{}) {
	for k, v := range funcs {
		t.FuncMap[k] = v
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

func TemplateData(ctx *Ctx, any interface{}) *TData {
	return &TData{
		Any:     any,
		Request: ctx.Request,
		Session: ctx.Session,
		Data:    ctx.Data,
		Flash:   allflashmessages(ctx),
	}
}
