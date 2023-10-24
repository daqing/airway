find /sql/*.sql | xargs -I{} psql -U $POSTGRES_USER -d $POSTGRES_DB -f {}
