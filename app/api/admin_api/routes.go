package admin_api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/daqing/airway/app/middleware"
	"github.com/daqing/airway/app/modules/rbac"
	"github.com/gin-gonic/gin"
)

type API struct{ service *rbac.Service }

func Routes(r *gin.RouterGroup, service *rbac.Service) {
	a := &API{service: service}
	r.GET("/admins", middleware.RequirePermission(service, "admins:read"), a.listAdmins)
	r.POST("/admins", middleware.RequirePermission(service, "admins:create"), a.createAdmin)
	r.PATCH("/admins/:id", middleware.RequirePermission(service, "admins:update"), a.updateAdmin)
	r.GET("/admins/:id/roles", middleware.RequirePermission(service, "admins:read"), a.adminRoles)
	r.PUT("/admins/:id/roles", middleware.RequirePermission(service, "admins:assign_roles"), a.assignRoles)
	r.GET("/roles", middleware.RequirePermission(service, "roles:read"), a.listRoles)
	r.POST("/roles", middleware.RequirePermission(service, "roles:create"), a.createRole)
	r.PATCH("/roles/:id", middleware.RequireSuperAdmin(), a.updateRole)
	r.GET("/roles/:id/permissions", middleware.RequirePermission(service, "roles:read"), a.rolePermissions)
	r.PUT("/roles/:id/permissions", middleware.RequirePermission(service, "roles:assign_permissions"), a.assignPermissions)
	r.GET("/permissions", middleware.RequirePermission(service, "permissions:read"), a.listPermissions)
	r.POST("/permissions", middleware.RequirePermission(service, "permissions:create"), a.createPermission)
	r.PATCH("/permissions/:id", middleware.RequireSuperAdmin(), a.updatePermission)
}

func (a *API) listAdmins(c *gin.Context) {
	v, e := a.service.ListAdmins(c)
	respond(c, v, e, http.StatusOK)
}
func (a *API) createAdmin(c *gin.Context) {
	var b struct {
		Login    string `json:"login"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if c.ShouldBindJSON(&b) != nil {
		fail(c, 400, "invalid_request", "invalid JSON body")
		return
	}
	v, e := a.service.CreateAdmin(c, b.Login, b.Email, b.Password)
	respond(c, v, e, 201)
}
func (a *API) updateAdmin(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var b struct {
		Email  string `json:"email"`
		Status string `json:"status"`
	}
	if c.ShouldBindJSON(&b) != nil {
		fail(c, 400, "invalid_request", "invalid JSON body")
		return
	}
	v, e := a.service.UpdateAdmin(c, id, b.Email, b.Status)
	respond(c, v, e, 200)
}
func (a *API) adminRoles(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	v, e := a.service.AdminRoles(c, id)
	respond(c, v, e, 200)
}
func (a *API) assignRoles(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var b struct {
		RoleIDs []int64 `json:"role_ids"`
	}
	if c.ShouldBindJSON(&b) != nil {
		fail(c, 400, "invalid_request", "invalid JSON body")
		return
	}
	e := a.service.AssignRoles(c, id, b.RoleIDs)
	if e == nil {
		v, x := a.service.AdminRoles(c, id)
		respond(c, v, x, 200)
		return
	}
	respond(c, nil, e, 200)
}
func (a *API) listRoles(c *gin.Context) { v, e := a.service.ListRoles(c); respond(c, v, e, 200) }
func (a *API) createRole(c *gin.Context) {
	var b struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if c.ShouldBindJSON(&b) != nil {
		fail(c, 400, "invalid_request", "invalid JSON body")
		return
	}
	v, e := a.service.CreateRole(c, b.Code, b.Name)
	respond(c, v, e, 201)
}
func (a *API) updateRole(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var b struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if c.ShouldBindJSON(&b) != nil {
		fail(c, 400, "invalid_request", "invalid JSON body")
		return
	}
	v, e := a.service.UpdateRole(c, id, b.Code, b.Name)
	respond(c, v, e, 200)
}
func (a *API) rolePermissions(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	v, e := a.service.RolePermissions(c, id)
	respond(c, v, e, 200)
}
func (a *API) assignPermissions(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var b struct {
		PermissionIDs []int64 `json:"permission_ids"`
	}
	if c.ShouldBindJSON(&b) != nil {
		fail(c, 400, "invalid_request", "invalid JSON body")
		return
	}
	e := a.service.AssignPermissions(c, id, b.PermissionIDs)
	if e == nil {
		v, x := a.service.RolePermissions(c, id)
		respond(c, v, x, 200)
		return
	}
	respond(c, nil, e, 200)
}
func (a *API) listPermissions(c *gin.Context) {
	v, e := a.service.ListPermissions(c)
	respond(c, v, e, 200)
}
func (a *API) createPermission(c *gin.Context) {
	var b struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if c.ShouldBindJSON(&b) != nil {
		fail(c, 400, "invalid_request", "invalid JSON body")
		return
	}
	v, e := a.service.CreatePermission(c, b.Code, b.Name)
	respond(c, v, e, 201)
}
func (a *API) updatePermission(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var b struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if c.ShouldBindJSON(&b) != nil {
		fail(c, 400, "invalid_request", "invalid JSON body")
		return
	}
	v, e := a.service.UpdatePermission(c, id, b.Code, b.Name)
	respond(c, v, e, 200)
}
func idParam(c *gin.Context) (int64, bool) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id < 1 {
		fail(c, 400, "invalid_id", "invalid resource id")
		return 0, false
	}
	return id, true
}
func respond(c *gin.Context, data any, err error, status int) {
	if err == nil {
		c.JSON(status, gin.H{"data": data, "error": nil})
		return
	}
	switch {
	case errors.Is(err, rbac.ErrLastSuperAdmin):
		fail(c, 409, "last_super_admin", err.Error())
	case errors.Is(err, rbac.ErrValidation):
		fail(c, 422, "validation_failed", err.Error())
	case errors.Is(err, rbac.ErrConflict), errors.Is(err, rbac.ErrAdminConflict):
		fail(c, 409, "conflict", err.Error())
	case errors.Is(err, rbac.ErrNotFound):
		fail(c, 404, "not_found", err.Error())
	default:
		fail(c, 500, "internal_error", "internal server error")
	}
}
func fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"data": nil, "error": gin.H{"code": code, "message": message, "request_id": c.GetHeader("X-Request-ID")}})
}
