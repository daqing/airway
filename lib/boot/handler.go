package boot

import (
	"bytes"
	"net/http"
	"strings"
)

// hideScrollbarsCSS hides scroll bars in the desktop WebView. Scrolling via
// wheel, trackpad and keyboard is unaffected; the bars are just not drawn,
// whatever the OS "show scroll bars" setting says.
const hideScrollbarsCSS = `<style>html{scrollbar-width:none}::-webkit-scrollbar{width:0;height:0;display:none}</style>`

// hideScrollbars injects the stylesheet into every HTML response. Non-HTML
// responses (JSON APIs, files, health checks) pass through untouched.
func hideScrollbars(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		iw := &htmlInjectWriter{ResponseWriter: w}
		next.ServeHTTP(iw, r)

		if !iw.html {
			return
		}

		body := iw.buf.String()
		if idx := strings.Index(body, "</head>"); idx >= 0 {
			body = body[:idx] + hideScrollbarsCSS + body[idx:]
		} else {
			body = hideScrollbarsCSS + body
		}
		_, _ = iw.ResponseWriter.Write([]byte(body))
	})
}

// htmlInjectWriter buffers text/html responses so the stylesheet can be
// spliced in; every other response streams straight through.
type htmlInjectWriter struct {
	http.ResponseWriter
	buf          bytes.Buffer
	html         bool
	headerPassed bool
}

func (w *htmlInjectWriter) WriteHeader(code int) {
	if w.headerPassed {
		return
	}
	w.headerPassed = true

	if strings.HasPrefix(w.Header().Get("Content-Type"), "text/html") {
		w.html = true
		// The injected stylesheet changes the body length.
		w.Header().Del("Content-Length")
	}

	w.ResponseWriter.WriteHeader(code)
}

func (w *htmlInjectWriter) Write(p []byte) (int, error) {
	if !w.headerPassed {
		w.WriteHeader(http.StatusOK)
	}
	if w.html {
		w.buf.Write(p)
		return len(p), nil
	}
	return w.ResponseWriter.Write(p)
}
