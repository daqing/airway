dev:
  overmind start -f Procfile.dev

setup:
  go install github.com/cosmtrek/air@latest
  brew install tmux
  brew install overmind
  curl -fsSL https://bun.sh/install | bash

cli:
  cd ./cli && go build && cd ..
  mv ./cli/cli ./bin/

build:
  GOOS=linux GOARCH=amd64 go build -o ./bin .

docker: build
  docker build -t airway-api -f Dockerfile --platform linux/amd64 .

dbdocker:
  docker build -f Dockerfile.db -t airway-db --platform linux/amd64 .

migrate:
  find db/*.sql | xargs -I{} psql -U $POSTGRES_USER -d airway -f {}

createdb:
  psql -U $POSTGRES_USER -d postgres -c "create database airway"

dropdb:
  psql -U $POSTGRES_USER -d postgres -c "drop database airway"

reset: dropdb createdb migrate

db:
  psql -U $POSTGRES_USER -d airway
