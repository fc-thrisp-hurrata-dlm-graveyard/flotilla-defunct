package flotilla

import (
	"os"
	"path/filepath"
)

type (
	// Staticor is an interface to handling static files requiring methods to
	// get & set string directories as well as determine(and potentially handle
	// however appropriate) existence of static files given a string and a Ctx.
	Staticor interface {
		StaticDirs(...string) []string
		Exists(string, *Ctx) bool
	}

	staticor struct {
		staticDirs []string
	}
)

func (env *Env) StaticorInit() {
	if env.Staticor == nil {
		env.Staticor = NewStaticor(env)
	}
}

// A string array of static dirs set in env.Store["staticdirectories"]
func (env *Env) StaticDirs(dirs ...string) []string {
	storedirs := env.Store["STATIC_DIRECTORIES"].List(dirs...)
	if env.Staticor != nil {
		return env.Staticor.StaticDirs(storedirs...)
	}
	return storedirs
}

func NewStaticor(env *Env) *staticor {
	s := &staticor{}
	s.StaticDirs(env.Store["STATIC_DIRECTORIES"].List()...)
	return s
}

func (s *staticor) StaticDirs(dirs ...string) []string {
	for _, dir := range dirs {
		s.staticDirs = doAdd(dir, s.staticDirs)
	}
	return s.staticDirs
}

func appStaticFile(requested string, ctx *Ctx) bool {
	exists := false
	for _, dir := range ctx.App.StaticDirs() {
		filepath.Walk(dir, func(path string, _ os.FileInfo, _ error) (err error) {
			if filepath.Base(path) == requested {
				f, _ := os.Open(path)
				ctx.ServeFile(f)
				exists = true
			}
			return err
		})
	}
	return exists
}

func appAssetFile(requested string, ctx *Ctx) bool {
	exists := false
	f, err := ctx.App.Assets.Get(requested)
	if err == nil {
		ctx.ServeFile(f)
		exists = true
	}
	return exists
}

func (s *staticor) Exists(requested string, ctx *Ctx) bool {
	exists := appStaticFile(requested, ctx)
	if !exists {
		exists = appAssetFile(requested, ctx)
	}
	return exists
}

func handleStatic(ctx *Ctx) {
	requested := filepath.Base(ctx.Request.URL.Path)
	exists := ctx.App.Staticor.Exists(requested, ctx)
	if !exists {
		ctx.Abort(404)
	} else {
		ctx.rw.WriteHeaderNow()
	}
}
