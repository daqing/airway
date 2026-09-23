package cmd

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"testing"
)

func TestScaffoldGeneratesStaticExport(t *testing.T) {
	useTempWorkingDir(t)

	if err := generateScaffold([]string{"post", "title:string"}); err != nil {
		t.Fatalf("generateScaffold: %v", err)
	}

	exportRaw := readFile(t, "export_posts.go")
	assertContains(t, exportRaw, "package main")
	assertContains(t, exportRaw, `static.Page{Slug: "/posts", Component: posts.Index()}`)
	assertContains(t, exportRaw, "cmd.SetStaticPagesProvider(")
	assertContains(t, exportRaw, "repo.FindAll[models.Post]()")
	assertContains(t, exportRaw, "posts.Show(item)")
	assertContains(t, exportRaw, `fmt.Sprintf("/posts/%d", item.ID)`)

	showRaw := readFile(t, filepath.Join("app", "views", "posts", "show.templ"))
	assertContains(t, showRaw, "templ Show(item *models.Post)")
	assertContains(t, showRaw, "{ item.Title }")
	assertContains(t, showRaw, `href="/posts"`)

	actionRaw := readFile(t, filepath.Join("app", "api", "posts_api", "posts_action.go"))
	assertContains(t, actionRaw, "func ShowAction(c *gin.Context)")
	assertContains(t, actionRaw, "repo.FindByID[models.Post]")

	routesRaw := readFile(t, filepath.Join("app", "api", "posts_api", "routes.go"))
	assertContains(t, routesRaw, `r.GET("/posts/:id", ShowAction)`)

	// Generated Go sources must at least parse.
	fset := token.NewFileSet()
	for _, path := range []string{
		"export_posts.go",
		filepath.Join("app", "api", "posts_api", "posts_action.go"),
		filepath.Join("app", "api", "posts_api", "routes.go"),
		filepath.Join("app", "services", "post.go"),
	} {
		if _, err := parser.ParseFile(fset, path, nil, parser.AllErrors); err != nil {
			t.Fatalf("%s does not parse: %v", path, err)
		}
	}
}
