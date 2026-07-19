package resource_definition_api

import (
	"errors"
	"strconv"

	"github.com/daqing/airway/app/middleware"
	"github.com/daqing/airway/app/modules/dynamicresource"
	"github.com/gin-gonic/gin"
)

type API struct{ service *dynamicresource.Service }
type request struct {
	Code      string                 `json:"code"`
	Name      string                 `json:"name"`
	TableName string                 `json:"table_name"`
	Schema    dynamicresource.Schema `json:"schema"`
}

func Routes(r *gin.RouterGroup, service *dynamicresource.Service) {
	a := &API{service: service}
	super := middleware.RequireSuperAdmin()
	r.GET("/resource-definitions", super, a.list)
	r.POST("/resource-definitions", super, a.create)
	r.GET("/resource-definitions/:id", super, a.get)
	r.PATCH("/resource-definitions/:id", super, a.update)
	r.POST("/resource-definitions/:id/validate", super, a.validate)
	r.POST("/resource-definitions/:id/publish", super, a.publish)
	r.POST("/resource-definitions/:id/deactivate", super, a.deactivate)
	r.GET("/resource-definitions/:id/versions", super, a.versions)
}
func (a *API) list(c *gin.Context) { v, e := a.service.List(c); respond(c, v, e, 200) }
func (a *API) get(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	v, e := a.service.Get(c, id)
	respond(c, v, e, 200)
}
func (a *API) create(c *gin.Context) {
	var b request
	if c.ShouldBindJSON(&b) != nil {
		fail(c, 400, "invalid_request", "invalid JSON body", nil)
		return
	}
	v, e := a.service.Create(c, b.Code, b.Name, b.TableName, b.Schema)
	respond(c, v, e, 201)
}
func (a *API) update(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	var b request
	if c.ShouldBindJSON(&b) != nil {
		fail(c, 400, "invalid_request", "invalid JSON body", nil)
		return
	}
	v, e := a.service.Update(c, id, b.Code, b.Name, b.TableName, b.Schema)
	respond(c, v, e, 200)
}
func (a *API) validate(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	v, e := a.service.Validate(c, id)
	respond(c, v, e, 200)
}
func (a *API) publish(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	admin, exists := middleware.CurrentAdmin(c)
	if !exists {
		fail(c, 401, "unauthenticated", "authentication required", nil)
		return
	}
	v, e := a.service.Publish(c, id, admin.ID)
	respond(c, v, e, 200)
}
func (a *API) deactivate(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	v, e := a.service.Deactivate(c, id)
	respond(c, v, e, 200)
}
func (a *API) versions(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	v, e := a.service.Versions(c, id)
	respond(c, v, e, 200)
}
func idParam(c *gin.Context) (int64, bool) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id < 1 {
		fail(c, 400, "invalid_id", "invalid resource id", nil)
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
	case errors.Is(err, dynamicresource.ErrNotFound):
		fail(c, 404, "not_found", err.Error(), nil)
	case errors.Is(err, dynamicresource.ErrConflict):
		fail(c, 409, "conflict", err.Error(), nil)
	case errors.Is(err, dynamicresource.ErrInvalidState):
		fail(c, 409, "invalid_state", err.Error(), nil)
	default:
		fail(c, 500, "internal_error", "internal server error", nil)
	}
}
func fail(c *gin.Context, status int, code, message string, details any) {
	errorBody := gin.H{"code": code, "message": message, "request_id": c.GetHeader("X-Request-ID")}
	if details != nil {
		errorBody["details"] = details
	}
	c.JSON(status, gin.H{"data": nil, "error": errorBody})
}
