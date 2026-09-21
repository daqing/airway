// Package boot wires a complete Airway runtime from explicit configuration.
// The web server reads its settings from the environment (see the root
// main.go); desktop wrappers need fixed, programmatic settings, so boot takes
// them as parameters instead of environment variables — the utils env
// snapshot gives process-start values priority over os.Setenv.
package boot

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/daqing/airway/app/websocket"
	"github.com/daqing/airway/lib/app"
	"github.com/daqing/airway/lib/migrate"
	"github.com/daqing/airway/lib/plugin"
	"github.com/daqing/airway/lib/redis_client"
	"github.com/daqing/airway/lib/repo"
	"github.com/daqing/airway/lib/storage"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

type Options struct {
	// AppName names the app in logs and startup output.
	AppName string
	// Env pins AIRWAY_ENV for the process ("" keeps the current value).
	// Anything but "local" runs Gin in release mode and serves the embedded
	// frontend bundle.
	Env string
	// DSN configures the database; empty skips DB setup entirely.
	DSN string
	// Migrations, when set, are applied to the database before serving.
	Migrations fs.FS
	// StorageRoot points the local file-storage driver at a directory;
	// empty keeps the storage package's environment-based configuration.
	StorageRoot string
	// RedisURL sets up Redis; empty skips Redis (desktop default).
	RedisURL string
	// StrictCORS replaces the permissive web CORS with same-origin-only.
	StrictCORS bool
	// SameOriginWS restricts WebSocket upgrades to same-origin requests.
	SameOriginWS bool
	// HideScrollbars injects a stylesheet into HTML responses so the
	// desktop WebView draws no scroll bars (scrolling still works).
	HideScrollbars bool
	// Routes mounts the application's public routes; it defaults to the
	// framework's built-in config.Routes. Desktop wrappers MUST pass the
	// host project's config.Routes here, otherwise the binary serves the
	// framework's demo landing page instead of the project's own views.
	Routes func(*gin.Engine)
	// HealthRoutes mounts the internal health check; defaults to the
	// framework's config.HealthRoutes.
	HealthRoutes func(*gin.Engine)
	// BootPlugins runs plugin.BootAll for the plugins compiled into the
	// binary.
	BootPlugins bool
	// Out receives migration progress; defaults to os.Stdout.
	Out io.Writer
}

// New boots the full stack — env, migrations, database, Redis, storage,
// plugins — and returns the app ready to serve via Handler().
func New(opts Options) (*app.App, error) {
	if opts.Env != "" {
		if err := os.Setenv("AIRWAY_ENV", opts.Env); err != nil {
			return nil, fmt.Errorf("pin AIRWAY_ENV: %w", err)
		}
	}

	appConfig := utils.AppConfig()
	if appConfig.Env == "" {
		return nil, errors.New("boot: AIRWAY_ENV is not set")
	}

	if !appConfig.IsLocal {
		gin.SetMode(gin.ReleaseMode)
	}

	if opts.Migrations != nil && opts.DSN != "" {
		if err := migrate.Run(migrate.Options{
			DSN:        opts.DSN,
			Migrations: opts.Migrations,
			Out:        opts.Out,
		}); err != nil {
			return nil, fmt.Errorf("boot: migrate: %w", err)
		}
	}

	if opts.DSN != "" {
		if _, err := repo.SetupDB(opts.DSN); err != nil {
			return nil, fmt.Errorf("boot: database setup: %w", err)
		}
	}

	if opts.RedisURL != "" {
		redis_client.Setup(opts.RedisURL)
	}

	storageCfg := storage.FromEnv()
	if opts.StorageRoot != "" {
		storageCfg.Driver = storage.DriverLocal
		storageCfg.Root = opts.StorageRoot
	}
	if _, err := storage.Setup(storageCfg); err != nil {
		return nil, fmt.Errorf("boot: storage setup: %w", err)
	}

	if opts.SameOriginWS {
		websocket.CheckSameOrigin()
	}

	if opts.BootPlugins {
		if err := plugin.BootAll(); err != nil {
			return nil, fmt.Errorf("boot: plugin boot: %w", err)
		}
	}

	var appOpts []app.Option
	if opts.StrictCORS {
		appOpts = append(appOpts, app.WithCORS(app.StrictOrigin()))
	}
	if opts.HideScrollbars {
		appOpts = append(appOpts, app.WithHandlerWrapper(hideScrollbars))
	}
	if opts.Routes != nil || opts.HealthRoutes != nil {
		appOpts = append(appOpts, app.WithRoutes(opts.Routes, opts.HealthRoutes))
	}

	return app.NewApp(opts.AppName, "0", appOpts...), nil
}
