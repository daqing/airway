dev:
  overmind start -f Procfile.dev

compose:
  docker-compose up --build

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
  go tool templ generate -path app/views -watch

# Format the .templ view sources (use `go tool templ fmt -fail app/views` in CI).
templ-fmt:
  go tool templ fmt app/views

# Bundle the frontend (app/assets/js -> app/assets/dist), then compile the binary.
build:
  go run . js:build
  go build -o bin/airway .

# Build the desktop target (./desktop) for the current OS via Wails v3.
# Requires `airway desktop:init` + the wails3 CLI; see docs/desktop.md.
desktop:
  cd desktop && wails3 task build

# Package the desktop target (.app / NSIS / deb+rpm) via Wails v3.
desktop-package:
  cd desktop && wails3 task package

build-on-mac:
  GOOS=linux GOARCH=amd64 go build .
  podman build -t airway -f Dockerfile.mac

