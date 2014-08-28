package flotilla

import (
	"net/http"
	"os"
	"path/filepath"
)

func engineStaticFile(dirs []string, requested string) (file http.File, exists bool) {
	file, exists = file, false
	for _, dir := range dirs {
		filepath.Walk(dir, func(path string, _ os.FileInfo, _ error) (err error) {
			if filepath.Base(path) == requested {
				f, _ := os.Open(path)
				file, exists = f, true
				//fmt.Printf("\n{path: %+v, base: %+v, file: %+v, requested: %+v, exists: %t}\n", path, filepath.Base(path), file, requested, exists)
			}
			return err
		})

	}
	return file, exists
}

func engineAssetFile(requested string, engine *Engine) (file http.File, exists bool) {
	file, exists = file, false
	f, err := engine.Assets.Get(requested)
	if err == nil {
		file, exists = f, true
		//fmt.Printf("\n\n{requested: %+v, file: %+v, exists: %t}\n", requested, file, exists)
	}
	return file, exists
}

func handleStatic(c *Ctx) {
	var f http.File
	var exists bool = false
	requested := filepath.Base(c.Request.URL.Path)
	f, exists = engineStaticFile(c.Engine.StaticDirs(), requested)
	if !exists {
		f, exists = engineAssetFile(requested, c.Engine)
	}
	if exists {
		c.ServeFile(f)
	} else {
		c.Abort(404)
	}
}
