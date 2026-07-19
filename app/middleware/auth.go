package middleware

import (
	"net/http"
	"strings"

	"github.com/daqing/airway/app/modules/identity"
	"github.com/daqing/airway/app/modules/rbac"
	"github.com/gin-gonic/gin"
)

const (
	adminContextKey = "authenticated_admin"
	sessionCookie   = "airway_session"
)

// Authenticate rejects requests without a valid administrator session.
func Authenticate(service *identity.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, err := service.Current(c, sessionToken(c))
		if err != nil {
			abort(c, http.StatusUnauthorized, "unauthenticated", "authentication required")
			return
		}
		c.Set(adminContextKey, admin)
		c.Next()
	}
}

// RequireSuperAdmin rejects authenticated administrators lacking elevated access.
// This is the Step 1 authorization boundary until granular RBAC is introduced.
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, ok := CurrentAdmin(c)
		if !ok || !admin.SuperAdmin {
			abort(c, http.StatusForbidden, "forbidden", "permission denied")
			return
		}
		c.Next()
	}
}

// RequirePermission authorizes against the current database state on every request.
func RequirePermission(service *rbac.Service, code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		admin, ok := CurrentAdmin(c)
		if !ok {
			abort(c, http.StatusUnauthorized, "unauthenticated", "authentication required")
			return
		}
		allowed, err := service.Allowed(c, admin, code)
		if err != nil {
			abort(c, http.StatusInternalServerError, "authorization_failed", "authorization check failed")
			return
		}
		if !allowed {
			abort(c, http.StatusForbidden, "forbidden", "permission denied")
			return
		}
		c.Next()
	}
}

// CurrentAdmin returns the administrator stored by Authenticate.
func CurrentAdmin(c *gin.Context) (identity.Admin, bool) {
	value, ok := c.Get(adminContextKey)
	if !ok {
		return identity.Admin{}, false
	}
	admin, ok := value.(identity.Admin)
	return admin, ok
}

func sessionToken(c *gin.Context) string {
	if cookie, err := c.Cookie(sessionCookie); err == nil {
		return cookie
	}
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if len(header) < len("Bearer ") || !strings.EqualFold(header[:len("Bearer ")], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(header[len("Bearer "):])
}

func abort(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"data": nil,
		"error": gin.H{
			"code":       code,
			"message":    message,
			"request_id": c.GetHeader("X-Request-ID"),
		},
	})
}
