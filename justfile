dev:
  overmind start -f Procfile.dev

install-deps:
  go install github.com/air-verse/air@latest
  brew install tmux
  brew install overmind

# Regenerate *_templ.go from the .templ views under app/views.
# Runs the pinned CLI via the //go:generate directive in generate.go.
generate:
  go generate ./...

# Regenerate the views, then keep them fresh while you edit .templ files.
generate-watch:
  go tool templ generate -watch

# Format the .templ view sources (use `go tool templ fmt -fail app/views` in CI).
templ-fmt:
  go tool templ fmt app/views

docker:
  docker build -t airway .
