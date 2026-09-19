package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/daqing/airway/lib/openapi"
	"github.com/daqing/airway/lib/utils"
	"github.com/gin-gonic/gin"
)

// runCLIOpenAPIGenerate writes the OpenAPI 3.2 document for the routes
// compiled into this binary (including enabled plugins) to ./openapi.json.
// The route table comes from the route source registered by the project's
// config package, so a host project documents its own API.
func runCLIOpenAPIGenerate(args []string) error {
	out := "openapi.json"

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case isHelpArg(arg):
			printCLIOpenAPIGenerateUsage(os.Stdout)
			return nil
		case arg == "--out" && i+1 < len(args):
			out = args[i+1]
			i++
		case strings.HasPrefix(arg, "--out="):
			out = strings.TrimPrefix(arg, "--out=")
		default:
			return fmt.Errorf("unknown argument: %s", arg)
		}
	}

	if strings.TrimSpace(out) == "" {
		return fmt.Errorf("--out must not be empty")
	}

	setup := openapi.RouteSource()
	if setup == nil {
		return fmt.Errorf("no route table compiled into this binary; run openapi:generate inside a project (go run . openapi:generate)")
	}

	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	setup(engine)

	body, err := openapi.Build(engine, openapi.BuildOptions{
		Version: Version,
		Server:  "http://localhost:" + cliPort() + utils.URLPrefix(),
		Warn:    log.Printf,
	})
	if err != nil {
		return err
	}

	if dir := filepath.Dir(out); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	if err := os.WriteFile(out, body, 0o644); err != nil {
		return err
	}

	fmt.Printf("OpenAPI document written to %s\n", out)
	return nil
}

// cliPort mirrors the server's port resolution (AIRWAY_PORT, then PORT) for
// the generated servers entry.
func cliPort() string {
	if port := utils.GetEnvMulti("AIRWAY_PORT", "PORT"); port != "" {
		return port
	}

	return "1900"
}

func printCLIOpenAPIGenerateUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage:")
	_, _ = fmt.Fprintln(w, "  airway openapi:generate [--out path]   write the OpenAPI 3.2 document (default ./openapi.json)")
}
