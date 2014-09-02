package flotilla

import (
	"os"
	"path/filepath"
)

func engineStaticFile(requested string, c *Ctx) (exists bool) {
	exists = false
	for _, dir := range c.Engine.StaticDirs() {
		filepath.Walk(dir, func(path string, _ os.FileInfo, _ error) (err error) {
			if filepath.Base(path) == requested {
				f, _ := os.Open(path)
				c.ServeFile(f)
				exists = true
			}
			return err
		})

	}
	return exists
}

func engineAssetFile(requested string, c *Ctx) (exists bool) {
	exists = false
	f, err := c.Engine.Assets.Get(requested)
	if err == nil {
		c.ServeFile(f)
		exists = true
	}
	return exists
}

func handleStatic(c *Ctx) {
	var exists bool = false
	requested := filepath.Base(c.Request.URL.Path)
	exists = engineStaticFile(requested, c)
	if !exists {
		exists = engineAssetFile(requested, c)
	}
	if !exists {
		c.Abort(404)
	}
}
