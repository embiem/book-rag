# Book RAG

## Get started

3. activate mise in the project (pins a go version & loads env vars)
4. `docker compose up -d` to run the db. Only for local development.
5. install ollama & `ollama pull embeddinggemma` for generating vector embeddings

Finally, use the following REST API endpoints to interact with the server.
You can use test books from /books, or download more from [https://www.gutenberg.org/](https://www.gutenberg.org/).

### REST API

- `GET /` - API Info
- `GET /books` - List available books for querying
- `POST /books` - Ingest a new book into the vector database (upload .txt file)
- `GET /books/{bookID}` - Query for snippets from a specific book

## DB

Using golang-migrate for migrations ([Tutorial](https://github.com/golang-migrate/migrate/blob/master/database/postgres/TUTORIAL.md)) and sqlc for queries, mutations & codegen ([Tutorial](https://docs.sqlc.dev/en/stable/tutorials/getting-started-postgresql.html)).

A PostgreSQL instance with pgvector installed is set-up using docker-compose.

[sqlc doc](https://docs.sqlc.dev/en/stable/howto/ddl.html) about handling SQL migrations.

### Generate client db code

Use the Makefile: `make generate`. Only necessary after changing `db/query.sql`.

Alternatively, run the command manually:

- sqlc codegen: `rm -rf data/ && sqlc generate`

### Migrations

1. Create Migration files: `migrate create -ext sql -dir db/migrations -seq your_migration_description`
2. Write the migrations in the created up & down files using SQL
3. Run up migrations: `migrate -database ${POSTGRESQL_URL} -path db/migrations up`
4. Check db & run down migrations to test they work as well: `migrate -database ${POSTGRESQL_URL} -path internal/db/migrations down` & check db as well
5. run up migrations again

Optionally, test migrations up & down on a separate local db instance e.g. by spinning up a stack with different name: `docker compose -p dbmigrations-testing up -d`.

When db is dirty, force db to a version reflecting it's real state: `migrate -database ${POSTGRESQL_URL} -path internal/db/migrations force VERSION`. Only for local development.

The current setup will run outstanding migrations at runtime on startup via `db/init.go`.

## TODO

- implement flow:
  - user uploads a .txt file of a book
  - simple chunking & vector embedding creation
  - insert into pgvector

- implement querying:
  - user sends a query & bookID
  - create vector embedding of query
  - perform similarity search on pgvector for that bookID
  - return 20 most similar snippets, sorted

- call LLM with a user's query + related snippets from the book

## Further Improvements

- ollama as a component in docker-compose with necessary embedding model
  pre-installed. So we don't have to require manually installing it.
- generate embeddings in batches & in parallel in rag/embedding.go
