package dynamic_resource_api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/daqing/airway/app/middleware"
	"github.com/daqing/airway/app/modules/dynamicresource"
	"github.com/daqing/airway/app/modules/rbac"
	"github.com/gin-gonic/gin"
)

type API struct {
	resources     *dynamicresource.Service
	authorization *rbac.Service
}

func Routes(r *gin.RouterGroup, resources *dynamicresource.Service, authorization *rbac.Service) {
	a := &API{resources: resources, authorization: authorization}
	r.GET("/resources/:resource/schema", a.schema)
	r.GET("/resources/:resource/records", a.list)
	r.POST("/resources/:resource/records", a.create)
	r.GET("/resources/:resource/records/:id", a.get)
	r.PATCH("/resources/:resource/records/:id", a.update)
	r.DELETE("/resources/:resource/records/:id", a.delete)
}
func (a *API) definition(c *gin.Context, action string) (dynamicresource.Definition, bool) {
	item, err := a.resources.Published(c, c.Param("resource"))
	if err != nil {
		respond(c, nil, err, 200)
		return item, false
	}
	admin, ok := middleware.CurrentAdmin(c)
	if !ok {
		fail(c, 401, "unauthenticated", "authentication required", nil)
		return item, false
	}
	permission := map[string]string{"list": item.Schema.Permissions.List, "read": item.Schema.Permissions.Read, "create": item.Schema.Permissions.Create, "update": item.Schema.Permissions.Update, "delete": item.Schema.Permissions.Delete}[action]
	allowed, err := a.authorization.Allowed(c, admin, permission)
	if err != nil {
		fail(c, 500, "authorization_failed", "authorization check failed", nil)
		return item, false
	}
	if !allowed {
		fail(c, 403, "forbidden", "permission denied", nil)
		return item, false
	}
	return item, true
}
func (a *API) schema(c *gin.Context) {
	item, ok := a.definition(c, "list")
	if !ok {
		return
	}
	respond(c, item, nil, 200)
}
func (a *API) list(c *gin.Context) {
	item, ok := a.definition(c, "list")
	if !ok {
		return
	}
	v, e := a.resources.ListRecords(c, item)
	respond(c, v, e, 200)
}
func (a *API) get(c *gin.Context) {
	item, ok := a.definition(c, "read")
	if !ok {
		return
	}
	id, ok := idParam(c)
	if !ok {
		return
	}
	v, e := a.resources.GetRecord(c, item, id)
	respond(c, v, e, 200)
}
func (a *API) create(c *gin.Context) {
	item, ok := a.definition(c, "create")
	if !ok {
		return
	}
	var body map[string]any
	if c.ShouldBindJSON(&body) != nil {
		fail(c, 400, "invalid_request", "invalid JSON body", nil)
		return
	}
	v, e := a.resources.CreateRecord(c, item, body)
	respond(c, v, e, 201)
}
func (a *API) update(c *gin.Context) {
	item, ok := a.definition(c, "update")
	if !ok {
		return
	}
	id, ok := idParam(c)
	if !ok {
		return
	}
	var body map[string]any
	if c.ShouldBindJSON(&body) != nil {
		fail(c, 400, "invalid_request", "invalid JSON body", nil)
		return
	}
	v, e := a.resources.UpdateRecord(c, item, id, body)
	respond(c, v, e, 200)
}
func (a *API) delete(c *gin.Context) {
	item, ok := a.definition(c, "delete")
	if !ok {
		return
	}
	id, ok := idParam(c)
	if !ok {
		return
	}
	e := a.resources.DeleteRecord(c, item, id)
	if e == nil {
		c.Status(http.StatusNoContent)
		return
	}
	respond(c, nil, e, 200)
}
func idParam(c *gin.Context) (int64, bool) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id < 1 {
		fail(c, 400, "invalid_id", "invalid record id", nil)
		return 0, false
	}
	return id, true
}
func respond(c *gin.Context, data any, err error, status int) {
	if err == nil {
		c.JSON(status, gin.H{"data": data, "error": nil})
		return
	}
	var validation dynamicresource.ValidationErrors
	switch {
	case errors.As(err, &validation):
		fail(c, 422, "validation_failed", err.Error(), validation)
	case errors.Is(err, dynamicresource.ErrRecordNotFound), errors.Is(err, dynamicresource.ErrNotFound):
		fail(c, 404, "not_found", err.Error(), nil)
	case errors.Is(err, dynamicresource.ErrConflict):
		fail(c, 409, "conflict", err.Error(), nil)
	default:
		fail(c, 500, "internal_error", "internal server error", nil)
	}
}
func fail(c *gin.Context, status int, code, message string, details any) {
	body := gin.H{"code": code, "message": message, "request_id": c.GetHeader("X-Request-ID")}
	if details != nil {
		body["details"] = details
	}
	c.JSON(status, gin.H{"data": nil, "error": body})
}
