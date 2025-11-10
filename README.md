# Book RAG

## Get started

3. `cp cmd/server/.env.example cmd/server/.env` & fill in env vars
4. `docker compose up -d` to run the db. Only for local development.
5. install ollama & `ollama pull mxbai-embed-large` for generating vector embeddings

## db

Using golang-migrate for migrations ([Tutorial](https://github.com/golang-migrate/migrate/blob/master/database/postgres/TUTORIAL.md)) and sqlc for queries, mutations & codegen ([Tutorial](https://docs.sqlc.dev/en/stable/tutorials/getting-started-postgresql.html)).

[sqlc doc](https://docs.sqlc.dev/en/stable/howto/ddl.html) about handling SQL migrations.

### Generate client code

Use the Makefile: `make generate`

Alternatively, run the command manually:

- sqlc codegen: `rm -rf data/ && sqlc generate`

### Migrations

For local dev, setup env var like so: `export POSTGRESQL_URL='postgres://postgres:password@localhost:5432/postgres?sslmode=disable'`.

Optionally, test migrations up & down on a separate local db instance e.g. by spinning up a stack with different name: `docker compose -p dbmigrations-testing up -d`.

1. Create Migration files: `migrate create -ext sql -dir db/migrations -seq your_migration_description`
2. Write the migrations in the created up & down files using SQL
3. Run up migrations: `migrate -database ${POSTGRESQL_URL} -path db/migrations up`
4. Check db & run down migrations to test they work as well: `migrate -database ${POSTGRESQL_URL} -path internal/db/migrations down` & check db as well
5. run up migrations again

When dirty, force db to a version reflecting it's real state: `migrate -database ${POSTGRESQL_URL} -path internal/db/migrations force VERSION`

## TODO

- add sqlc (with pgx)
- add docker compose setup for postgres using pgvector & ollama
- setup embedding model via ollama
- implement flow:
  - user uploads a .txt file of a book
  - simple chunking & vector embedding creation
  - insert into pgvector

- implement querying:
  - user sends a query & bookID
  - create vector embedding of query
  - perform similarity search on pgvector for that bookID
  - return 20 most similar snippets, sorted
