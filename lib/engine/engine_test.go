package engine

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeEngine struct {
	name      string
	mountPath string
	bootErr   error
	booted    bool
	models    map[string]any
}

func (e *fakeEngine) Name() string      { return e.name }
func (e *fakeEngine) MountPath() string { return e.mountPath }

func (e *fakeEngine) Routes(r *gin.RouterGroup) {
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, e.name+" pong")
	})
}

func (e *fakeEngine) Boot() error {
	e.booted = true
	return e.bootErr
}

func (e *fakeEngine) REPLModels() map[string]any { return e.models }

type plainEngine struct {
	name      string
	mountPath string
}

func (e *plainEngine) Name() string              { return e.name }
func (e *plainEngine) MountPath() string         { return e.mountPath }
func (e *plainEngine) Routes(r *gin.RouterGroup) {}

func TestRegisterAndFind(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	e := &fakeEngine{name: "im", mountPath: "/api/v1/im"}
	Register(e)

	if got := Find("im"); got != e {
		t.Fatalf("expected Find to return the registered engine, got %v", got)
	}
	if got := Find("missing"); got != nil {
		t.Fatalf("expected Find to return nil for unknown engine, got %v", got)
	}

	engines := Engines()
	if len(engines) != 1 || engines[0] != e {
		t.Fatalf("unexpected engines: %#v", engines)
	}
}

func TestRegisterDuplicateNamePanics(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	Register(&plainEngine{name: "im", mountPath: "/im"})

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate engine name")
		}
	}()

	Register(&plainEngine{name: "im", mountPath: "/other"})
}

func TestRegisterInvalidEnginePanics(t *testing.T) {
	cases := map[string]Engine{
		"nil":         nil,
		"empty name":  &plainEngine{name: "", mountPath: "/x"},
		"bad mount":   &plainEngine{name: "x", mountPath: "x"},
		"empty mount": &plainEngine{name: "x", mountPath: ""},
	}

	for label, e := range cases {
		t.Run(label, func(t *testing.T) {
			ResetForTest()
			t.Cleanup(ResetForTest)

			defer func() {
				if recover() == nil {
					t.Fatalf("expected panic for %s", label)
				}
			}()

			Register(e)
		})
	}
}

func TestMountAllServesEngineRoutes(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	gin.SetMode(gin.TestMode)

	Register(&fakeEngine{name: "im", mountPath: "/api/v1/im"})
	Register(&fakeEngine{name: "blog", mountPath: "/blog"})

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

func TestBootAllRunsBootableEnginesInOrder(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	first := &fakeEngine{name: "first", mountPath: "/first"}
	second := &plainEngine{name: "second", mountPath: "/second"}
	third := &fakeEngine{name: "third", mountPath: "/third"}

	Register(first)
	Register(second)
	Register(third)

	if err := BootAll(); err != nil {
		t.Fatalf("unexpected boot error: %v", err)
	}

	if !first.booted || !third.booted {
		t.Fatalf("expected bootable engines to boot, got first=%v third=%v", first.booted, third.booted)
	}
}

func TestBootAllStopsOnError(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	bootErr := errors.New("boom")
	failing := &fakeEngine{name: "failing", mountPath: "/failing", bootErr: bootErr}
	after := &fakeEngine{name: "after", mountPath: "/after"}

	Register(failing)
	Register(after)

	err := BootAll()
	if !errors.Is(err, bootErr) {
		t.Fatalf("expected boot error, got %v", err)
	}
	if after.booted {
		t.Fatal("expected boot to stop after the failing engine")
	}
}

func TestREPLNamespacesMergesEngineModels(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	Register(&fakeEngine{name: "im", mountPath: "/im", models: map[string]any{
		"Message": struct{}{},
	}})
	Register(&plainEngine{name: "blog", mountPath: "/blog"})

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

	Register(&fakeEngine{name: "im", mountPath: "/im", models: map[string]any{
		"User": struct{}{},
	}})

	if _, err := REPLNamespaces("User"); err == nil {
		t.Fatal("expected an error when an engine model conflicts with a host model")
	}
}
