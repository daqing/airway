if [[ -z "$1" ]]; then
  echo "usage: $0 [userName] [dbName]"
  exit
fi

if [[ -z "$2" ]]; then
  echo "usage: $0 [userName] [dbName]"
  exit
fi

name=$1
db=$2

psql -U $name -d postgres -c "create database ${db}"
