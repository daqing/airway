dev:
  overmind start -f Procfile.dev

install-deps:
  go install github.com/cosmtrek/air@latest
  brew install tmux
  brew install overmind
  curl -fsSL https://bun.sh/install | bash

build-cli:
  cd ./cli && go build && cd ..
  mv ./cli/cli ./bin/

build-cli-docker:
  cd ./cli && GOOS=linux GOARCH=amd64 go build -o ../bin/cli_amd .

cli +args:
  ./bin/cli {{args}}

packjs:
  cd ./core/frontend/javascripts/ && bun build --minify --splitting --outdir=../../public/js ./src/*.jsx

build: packjs
  GOOS=linux GOARCH=amd64 go build -o ./bin .

bun:
  cd ./core/frontend/javascripts && bun install

docker: build build-cli-docker
  docker build -t airway -f Dockerfile --platform linux/amd64 .

dbdocker:
  docker build -t airway_db -f Dockerfile.db  --platform linux/amd64 .

migrate:
  find db/*.sql | xargs -I{} psql -U $POSTGRES_USER -d airway -f {}

createdb:
  psql -U $POSTGRES_USER -d postgres -c "create database airway"

dropdb:
  psql -U $POSTGRES_USER -d postgres -c "drop database airway"

reset: dropdb createdb migrate

db:
  psql -U $POSTGRES_USER -d airway
