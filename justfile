dev:
  overmind start -f Procfile.dev

install-deps:
  go install github.com/cosmtrek/air@latest
  brew install tmux
  brew install overmind

docker:
  docker build -t airway .
