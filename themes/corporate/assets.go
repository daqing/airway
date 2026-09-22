package corporate

import (
	"embed"
	"io/fs"
)

//go:embed all:assets
var embeddedAssets embed.FS

// assetsFS is re-rooted at assets/, so site.Build copies its contents
// directly into the site output's assets/ directory.
var assetsFS = func() fs.FS {
	sub, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}()
