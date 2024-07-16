dev:
  rm .overmind.sock
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
  rm -f ./public/js/*
  cd ./app/frontend/javascripts/ && bun build --minify --splitting --outdir=../../public/js ./src/*.jsx

build: packjs
  GOOS=linux GOARCH=amd64 go build -o ./bin .

bun:
  cd ./app/frontend/javascripts && bun install

docker: build build-cli-docker
  docker build -t airway -f Dockerfile --platform linux/amd64 .

db-docker:
  docker build -t airway_db -f Dockerfile.db  --platform linux/amd64 .

push:
  docker tag airway reg.appmz.cn/daqing/airway
  docker push reg.appmz.cn/daqing/airway

  docker tag airway_db reg.appmz.cn/daqing/airway_db
  docker push reg.appmz.cn/daqing/airway_db

migrate:
  find db/*.sql | xargs -I{} psql -U $POSTGRES_USER -d airway -f {}

create-db:
  psql -U $POSTGRES_USER -d postgres -c "create database airway"

drop-db:
  psql -U $POSTGRES_USER -d postgres -c "drop database airway"

setup-db: create-db migrate
reset-db: drop-db create-db migrate

db:
  psql -U $POSTGRES_USER -d airway
