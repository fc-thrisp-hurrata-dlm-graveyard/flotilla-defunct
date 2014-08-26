package flotilla

import (
	"net/http"
	"os"
	"path/filepath"
)

func engineStaticFile(engine *Engine, requested string) (hasfile http.File, err error) {
	for _, dir := range engine.StaticDirs {
		filepath.Walk(dir, func(path string, _ os.FileInfo, _ error) error {
			if filepath.Base(path) == requested {
				hasfile, err = os.Open(path)
			}
			return nil
		})
	}
	return hasfile, err
}

func handleStatic(c *Context) {
	requested := filepath.Base(c.Request.URL.Path)
	hasfile, err := engineStaticFile(c.Engine, requested)
	if hasfile == nil {
		hasfile, err = c.Engine.Assets.Get(requested)
	}
	if err == nil {
		c.ServeFile(hasfile)
	} else {
		c.Abort(404)
	}
}
