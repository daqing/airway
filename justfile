dev:
  AIRWAY_ENV=local air

build:
  GOOS=linux GOARCH=amd64 go build -o ./bin .

docker: build
  docker build -t airway -f Dockerfile --platform linux/amd64 .

migrate:
  find db -name '*.sql' | sort | xargs -I{} psql -U daqing -d airway -f {}

createdb:
  psql -U daqing -d postgres -c "create database airway"

dropdb:
  psql -U daqing -d postgres -c "drop database airway"

reset: dropdb createdb migrate

db:
  psql -U daqing -d airway
