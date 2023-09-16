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

echo `pwd`

find . -name '*.sql' | sort | xargs -I{} psql -U $name -d $db -f {}
