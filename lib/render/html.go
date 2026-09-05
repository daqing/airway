package render

import (
	"bytes"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin"
)

// HTML renders a templ component as the response body with a 200 status,
// setting the Content-Type to text/html. This is the HTML counterpart to the
// JSON helpers above; pages are kept under app/views as .templ files and wired
// into actions like:
//
//	render.HTML(c, home.Index())
func HTML(c *gin.Context, comp templ.Component) {
	HTMLStatus(c, http.StatusOK, comp)
}

// HTMLStatus renders a templ component as the response body with the given
// status code. The component is rendered into a buffer first so that a render
// failure surfaces as a 500 instead of a half-written response.
func HTMLStatus(c *gin.Context, status int, comp templ.Component) {
	var buf bytes.Buffer
	if err := comp.Render(c.Request.Context(), &buf); err != nil {
		_ = c.Error(err)
		c.String(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
		c.Abort()
		return
	}

	c.Data(status, "text/html; charset=utf-8", buf.Bytes())
}
