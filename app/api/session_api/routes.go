package session_api

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/daqing/airway/app/modules/identity"
	"github.com/daqing/airway/lib/repo"
	"github.com/gin-gonic/gin"
)

const cookieName = "airway_session"

// API groups the identity service, rate limiter, and cookie settings used by session routes.
type API struct {
	service      *identity.Service
	limiter      *RateLimiter
	secureCookie bool
}

// Routes registers login, current session, and logout endpoints on the given router group.
func Routes(r *gin.RouterGroup) {
	db, ok := repo.CurrentDBOK()
	if !ok {
		return
	}
	api := &API{
		service:      identity.NewService(db, 12*time.Hour),
		limiter:      NewRateLimiter(5, 15*time.Minute),
		secureCookie: os.Getenv("AIRWAY_ENV") != "local" && os.Getenv("AIRWAY_ENV") != "test",
	}
	r.POST("/sessions", api.login)
	r.GET("/session", api.current)
	r.DELETE("/session", api.logout)
}

// loginRequest represents the payload accepted by the administrator login endpoint.
type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// login handles a login request and issues a secure session cookie to the client.
func (a *API) login(c *gin.Context) {
	var req loginRequest
	if c.ShouldBindJSON(&req) != nil {
		fail(c, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	key := c.ClientIP() + "|" + strings.ToLower(strings.TrimSpace(req.Login))
	if !a.limiter.Allow(key) {
		_ = a.service.Audit(c, nil, "auth.login_rate_limited", "session", "", "failure", c.ClientIP(), requestID(c), nil)
		fail(c, http.StatusTooManyRequests, "rate_limited", "too many login attempts")
		return
	}
	admin, token, expires, err := a.service.Login(c, req.Login, req.Password, c.ClientIP(), requestID(c))
	if err != nil {
		fail(c, http.StatusUnauthorized, "invalid_credentials", "invalid login or password")
		return
	}
	a.limiter.Reset(key)
	http.SetCookie(c.Writer, &http.Cookie{Name: cookieName, Value: token, Path: "/api/v1", HttpOnly: true, Secure: a.secureCookie, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: int(time.Until(expires).Seconds())})
	c.JSON(http.StatusCreated, gin.H{"data": admin, "error": nil})
}

// current returns the administrator associated with the active session.
func (a *API) current(c *gin.Context) {
	admin, err := a.service.Current(c, sessionToken(c))
	if err != nil {
		fail(c, http.StatusUnauthorized, "unauthenticated", "authentication required")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": admin, "error": nil})
}

// logout revokes the active session and clears the client session cookie.
func (a *API) logout(c *gin.Context) {
	if err := a.service.Logout(c, sessionToken(c), c.ClientIP(), requestID(c)); err != nil {
		fail(c, http.StatusUnauthorized, "unauthenticated", "authentication required")
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: cookieName, Value: "", Path: "/api/v1", HttpOnly: true, Secure: a.secureCookie, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	c.Status(http.StatusNoContent)
}

// sessionToken reads the session token from a cookie or falls back to the Authorization header.
func sessionToken(c *gin.Context) string {
	if cookie, err := c.Cookie(cookieName); err == nil {
		return cookie
	}
	return strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
}

// requestID reads the request identifier used for auditing and error tracing.
func requestID(c *gin.Context) string { return c.GetHeader("X-Request-ID") }

// fail returns the specified HTTP status using the standard error response structure.
func fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"data": nil, "error": gin.H{"code": code, "message": message, "request_id": requestID(c)}})
}
