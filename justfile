dev:
  overmind start -f Procfile.dev

install-deps:
  go install github.com/cosmtrek/air@latest
  brew install tmux
  brew install overmind

build:
  GOOS=linux GOARCH=amd64 go build -o ./bin .

docker: build
  docker build -t airway -f Dockerfile --platform linux/amd64 .
