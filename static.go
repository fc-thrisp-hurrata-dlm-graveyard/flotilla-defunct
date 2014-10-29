package flotilla

import (
	"os"
	"path/filepath"
)

func appStaticFile(requested string, r *R) (exists bool) {
	exists = false
	for _, dir := range r.App.StaticDirs() {
		filepath.Walk(dir, func(path string, _ os.FileInfo, _ error) (err error) {
			if filepath.Base(path) == requested {
				f, _ := os.Open(path)
				r.ServeFile(f)
				exists = true
			}
			return err
		})
	}
	return exists
}

func appAssetFile(requested string, r *R) (exists bool) {
	exists = false
	f, err := r.App.Assets.Get(requested)
	if err == nil {
		r.ServeFile(f)
		exists = true
	}
	return exists
}

func handleStatic(r *R) {
	var exists bool = false
	requested := filepath.Base(r.Request.URL.Path)
	exists = appStaticFile(requested, r)
	if !exists {
		exists = appAssetFile(requested, r)
	}
	if !exists {
		r.Abort(404)
	} else {
		r.rw.WriteHeaderNow()
	}
}
