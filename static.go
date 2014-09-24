package flotilla

import (
	"os"
	"path/filepath"
)

func engineStaticFile(requested string, r *R) (exists bool) {
	exists = false
	for _, dir := range r.app.StaticDirs() {
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

func engineAssetFile(requested string, r *R) (exists bool) {
	exists = false
	f, err := r.app.Assets.Get(requested)
	if err == nil {
		r.ServeFile(f)
		exists = true
	}
	return exists
}

func handleStatic(r *R) {
	var exists bool = false
	requested := filepath.Base(r.Request.URL.Path)
	exists = engineStaticFile(requested, r)
	if !exists {
		exists = engineAssetFile(requested, r)
	}
	if !exists {
		r.Abort(404)
	} else {
		r.rw.WriteHeaderNow()
	}
}
