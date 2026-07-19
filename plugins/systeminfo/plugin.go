// Package systeminfo is a minimal, statically linked example plugin showing
// permission, menu, and authenticated API registration through the public SDK.
package systeminfo

import (
	"net/http"
	"runtime"
	"time"

	pluginsdk "github.com/daqing/airway/plugin"
	"github.com/gin-gonic/gin"
)

const PermissionRead = "system_info:read"

type Plugin struct{}

func (Plugin) Manifest() pluginsdk.Manifest {
	return pluginsdk.Manifest{
		APIVersion: pluginsdk.APIVersion,
		Name:       "system-info",
		Version:    "1.0.0",
		Core:       ">=0.1.0 <0.2.0",
		Entry:      "systeminfo.Register",
		Permissions: []pluginsdk.Permission{
			{Code: PermissionRead, Name: "查看系统信息"},
		},
		Dependencies: []pluginsdk.Dependency{},
		Migrations:   []string{},
	}
}

func (Plugin) Register(ctx pluginsdk.Context) error {
	if err := ctx.AddPermission(pluginsdk.Permission{Code: PermissionRead, Name: "查看系统信息"}); err != nil {
		return err
	}
	if err := ctx.AddMenu(pluginsdk.Menu{Code: "system-info", Label: "系统信息", Path: "/plugins/system-info", Permission: PermissionRead}); err != nil {
		return err
	}
	return ctx.Handle(http.MethodGet, "/status", PermissionRead, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"status":      "ok",
				"plugin":      "system-info",
				"version":     "1.0.0",
				"go_version":  runtime.Version(),
				"server_time": time.Now().UTC(),
			},
			"error": nil,
		})
	})
}

func init() {
	if err := pluginsdk.Register(Plugin{}); err != nil {
		panic(err)
	}
}
