package main

import (
	"github.com/daqing/airway/app/views/home"
	"github.com/daqing/airway/cmd"
	"github.com/daqing/airway/lib/static"
)

// The framework's own static export: `airway static:build` renders these
// pages into dist/ together with the committed frontend bundle (see
// docs/static-export.md). Pages must render without a request — bake
// build-time data into the component instead of querying the database.
func init() {
	cmd.SetStaticPages(
		static.Page{Slug: "/", Component: home.Index(3)},
	)
}
