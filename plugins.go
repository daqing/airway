package main

// Plugins are optional feature modules shipped as independent Go modules
// (see docs/plugin.md). Enable one by adding a blank import below and
// running `go mod tidy`:
//
//	import (
//		_ "github.com/example/airway-im-plugin"
//	)
//
// The plugin's package init registers it with lib/plugin; the framework then
// mounts its routes, boots it, and exposes its migrations and REPL models.
