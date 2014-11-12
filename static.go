package flotilla

import (
	"os"
	"path/filepath"
)

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

func handleStatic(ctx *Ctx) {
	var exists bool = false
	requested := filepath.Base(ctx.Request.URL.Path)
	exists = appStaticFile(requested, ctx)
	if !exists {
		exists = appAssetFile(requested, ctx)
	}
	if !exists {
		ctx.Abort(404)
	} else {
		ctx.rw.WriteHeaderNow()
	}
}
