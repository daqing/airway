package assets

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"sync/atomic"

	"github.com/a-h/templ"

	"github.com/daqing/airway/lib/jsbuild"
	"github.com/daqing/airway/lib/utils"
)

// islandSeq numbers mount points uniquely within the process; the runtime
// pairs data-island-id with the #island-data-<id> JSON script.
var islandSeq atomic.Uint64

// Island renders an interactive island: a [data-island] mount point plus
// the initial props as an adjacent JSON script. name must match a file
// under app/assets/js/islands/ (default-exporting the component); the
// generated registry maps it to the component and the runtime mounts it
// with these props. Pages remain complete without JavaScript — the mount
// point renders empty and the props stay inert.
func Island(name string, props any) templ.Component {
	id := islandSeq.Add(1)
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
		mount := fmt.Sprintf(`<div data-island="%s" data-island-id="%s"></div>`+"\n",
			templ.EscapeString(name), strconv.FormatUint(id, 10))
		if _, err := io.WriteString(w, mount); err != nil {
			return err
		}
		return templ.JSONScript(fmt.Sprintf("island-data-%d", id), props).Render(ctx, w)
	})
}

// Scripts returns the tag that loads the island runtime and bundle: the dev
// server's in-memory output under AIRWAY_ENV=local, otherwise the embedded
// bundle behind its cache-busting ?v=<hash> query. Include it once per
// page — base.templ already does.
func Scripts() templ.Component {
	if utils.AppConfig().IsLocal {
		return templ.Raw(`<script type="module" src="` + utils.URLPrefix() + `/assets/` + jsbuild.EntryJS + `"></script>`)
	}
	return templ.Raw(`<script type="module" src="` + utils.URLPrefix() + EntryPath() + `"></script>`)
}

// Stylesheet returns the link tag for the bundled component styles
// (css/airway.css via the entry import). Same caching rules as Scripts.
func Stylesheet() templ.Component {
	if utils.AppConfig().IsLocal {
		return templ.Raw(`<link rel="stylesheet" href="` + utils.URLPrefix() + `/assets/` + jsbuild.EntryCSS + `">`)
	}
	m, err := Manifest()
	if err != nil || m.Hash == "" {
		return templ.Raw(`<link rel="stylesheet" href="` + utils.URLPrefix() + `/assets/` + jsbuild.EntryCSS + `">`)
	}
	return templ.Raw(`<link rel="stylesheet" href="` + utils.URLPrefix() + `/assets/` + jsbuild.EntryCSS + `?v=` + m.Hash + `">`)
}
