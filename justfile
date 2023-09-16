dev:
  AIRWAY_ENV=local air

setup:
  go install github.com/cosmtrek/air@latest

build:
  GOOS=linux GOARCH=amd64 go build -o ./bin .

docker: build
  docker build -t airway -f Dockerfile --platform linux/amd64 .
