dev:
  overmind start -f Procfile.dev

install-deps:
  go install github.com/cosmtrek/air@latest
  brew install tmux
  brew install overmind

build-cli:
  cd ./cli && go build && cd ..
  mv ./cli/cli ./bin/

build-cli-docker:
  cd ./cli && GOOS=linux GOARCH=amd64 go build -o ../bin/cli_amd .

cli +args:
  ./bin/cli {{args}}

build:
  GOOS=linux GOARCH=amd64 go build -o ./bin .

docker: build build-cli-docker
  docker build -t airway -f Dockerfile --platform linux/amd64 .
