package flotilla

import (
	"net/http"
	"os"
	"path/filepath"
)

func engineStaticFile(engine *Engine, requested string) (hasfile http.File, err error) {
	for _, dir := range engine.StaticPaths {
		filepath.Walk(dir, func(path string, _ os.FileInfo, _ error) error {
			if filepath.Base(path) == requested {
				hasfile, err = os.Open(path)
			}
			return nil
		})
	}
	return hasfile, err
}

func engineStaticAsset(flotilla []Flotilla, requested string) (hasfile http.File, err error) {
	for _, f := range flotilla {
		if f.HasAssets() {
			hasfile, err = f.GetAsset(requested)
			return hasfile, err
		}
	}
	return nil, newError("no matching asset file")
}

func handleStatic(c *Context) {
	requested := filepath.Base(c.Request.URL.Path)
	hasfile, err := engineStaticFile(c.Engine, requested)
	if hasfile == nil {
		hasfile, err = engineStaticAsset(c.Engine.Flotilla(), requested)
	}
	if err == nil {
		c.ServeAsset(hasfile)
	}
}
