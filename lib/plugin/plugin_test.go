package plugin

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakePlugin struct {
	name      string
	mountPath string
	bootErr   error
	booted    bool
	models    map[string]any
}

func (p *fakePlugin) Name() string      { return p.name }
func (p *fakePlugin) MountPath() string { return p.mountPath }

func (p *fakePlugin) Routes(r *gin.RouterGroup) {
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, p.name+" pong")
	})
}

func (p *fakePlugin) Boot() error {
	p.booted = true
	return p.bootErr
}

func (p *fakePlugin) REPLModels() map[string]any { return p.models }

type plainPlugin struct {
	name      string
	mountPath string
}

func (p *plainPlugin) Name() string              { return p.name }
func (p *plainPlugin) MountPath() string         { return p.mountPath }
func (p *plainPlugin) Routes(r *gin.RouterGroup) {}

func TestRegisterAndFind(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	p := &fakePlugin{name: "im", mountPath: "/api/v1/im"}
	Register(p)

	if got := Find("im"); got != p {
		t.Fatalf("expected Find to return the registered plugin, got %v", got)
	}
	if got := Find("missing"); got != nil {
		t.Fatalf("expected Find to return nil for unknown plugin, got %v", got)
	}

	plugins := Plugins()
	if len(plugins) != 1 || plugins[0] != p {
		t.Fatalf("unexpected plugins: %#v", plugins)
	}
}

func TestRegisterDuplicateNamePanics(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	Register(&plainPlugin{name: "im", mountPath: "/im"})

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate plugin name")
		}
	}()

	Register(&plainPlugin{name: "im", mountPath: "/other"})
}

func TestRegisterInvalidPluginPanics(t *testing.T) {
	cases := map[string]Plugin{
		"nil":         nil,
		"empty name":  &plainPlugin{name: "", mountPath: "/x"},
		"bad mount":   &plainPlugin{name: "x", mountPath: "x"},
		"empty mount": &plainPlugin{name: "x", mountPath: ""},
	}

	for label, p := range cases {
		t.Run(label, func(t *testing.T) {
			ResetForTest()
			t.Cleanup(ResetForTest)

			defer func() {
				if recover() == nil {
					t.Fatalf("expected panic for %s", label)
				}
			}()

			Register(p)
		})
	}
}

func TestMountAllServesPluginRoutes(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	gin.SetMode(gin.TestMode)

	Register(&fakePlugin{name: "im", mountPath: "/api/v1/im"})
	Register(&fakePlugin{name: "blog", mountPath: "/blog"})

	router := gin.New()
	MountAll(router)

	for _, path := range []string{"/api/v1/im/ping", "/blog/ping"} {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200, got %d", path, recorder.Code)
		}
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/im/ping", nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404 outside the mount path, got %d", recorder.Code)
	}
}

func TestBootAllRunsBootablePluginsInOrder(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	first := &fakePlugin{name: "first", mountPath: "/first"}
	second := &plainPlugin{name: "second", mountPath: "/second"}
	third := &fakePlugin{name: "third", mountPath: "/third"}

	Register(first)
	Register(second)
	Register(third)

	if err := BootAll(); err != nil {
		t.Fatalf("unexpected boot error: %v", err)
	}

	if !first.booted || !third.booted {
		t.Fatalf("expected bootable plugins to boot, got first=%v third=%v", first.booted, third.booted)
	}
}

func TestBootAllStopsOnError(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	bootErr := errors.New("boom")
	failing := &fakePlugin{name: "failing", mountPath: "/failing", bootErr: bootErr}
	after := &fakePlugin{name: "after", mountPath: "/after"}

	Register(failing)
	Register(after)

	err := BootAll()
	if !errors.Is(err, bootErr) {
		t.Fatalf("expected boot error, got %v", err)
	}
	if after.booted {
		t.Fatal("expected boot to stop after the failing plugin")
	}
}

func TestREPLNamespacesMergesPluginModels(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	Register(&fakePlugin{name: "im", mountPath: "/im", models: map[string]any{
		"Message": struct{}{},
	}})
	Register(&plainPlugin{name: "blog", mountPath: "/blog"})

	models, err := REPLNamespaces("User")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := models["Message"]; !ok {
		t.Fatalf("expected Message model, got %#v", models)
	}
}

func TestREPLNamespacesRejectsConflicts(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	Register(&fakePlugin{name: "im", mountPath: "/im", models: map[string]any{
		"User": struct{}{},
	}})

	if _, err := REPLNamespaces("User"); err == nil {
		t.Fatal("expected an error when a plugin model conflicts with a host model")
	}
}
