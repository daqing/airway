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

build:
  GOOS=linux GOARCH=amd64 go build -o ./bin .

docker: build build-cli-docker
  docker build -t airway -f Dockerfile --platform linux/amd64 .

push:
  docker tag airway reg.appmz.cn/daqing/airway
  docker push reg.appmz.cn/daqing/airway

create-db:
  psql -U $POSTGRES_USER -d postgres -c "create database airway"

drop-db:
  psql -U $POSTGRES_USER -d postgres -c "drop database airway"

setup-db: create-db
reset-db: drop-db create-db

db:
  psql -U $POSTGRES_USER -d airway
