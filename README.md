# Book RAG

## Prerequisites

- mise-en-place: <https://mise.jdx.dev/>
- docker

## Get started

1. activate mise in the project (pins a go version & loads env vars)
2. `docker compose up -d` to run the db. Only for local development.
3. install [ollama](https://ollama.com/download) & run `ollama pull embeddinggemma`
   to download the vector embeddings model used for generating vector embeddings

Finally, use the following REST API endpoints to interact with the server.
You can use test books from /books, or download more from [https://www.gutenberg.org/](https://www.gutenberg.org/).

### REST API

- `GET /` - API Info
- `GET /books` - List available books for querying
- `POST /books` - Ingest a new book into the vector database
  - Content-Type: `multipart/form-data`
  - Form field: `file`. Plain text file (.txt) containing the book content
  - Alternatively a `text` field with the book's text
  - Returns the newly created book ID
- `POST /books/{bookID}/query` - Query for snippets from a specific book
  - Request body: `{"query": "search text", "limit": 20}`
  - `query` (required): Search query text
  - `limit` (optional): Number of results to return (default: 20, max: 100)
  - Returns passages ranked by similarity with scores
- `POST /books/{bookID}/rag` - Provide a prompt and receive a LLM generated
  answer enriched with relevant passages from the book

#### Example curl commands

```bash
# Ingest a new book
curl -X POST http://localhost:3000/books \
  -F "file=@books/romeo_and_juliet.txt"

# List all books
curl http://localhost:8080/books

# Query a book (replace {bookID} with actual ID from previous commands)
curl -X POST http://localhost:8080/books/{bookID}/query \
  -H "Content-Type: application/json" \
  -d '{"query": "What happens in the balcony scene?"}'
```

## DB

Using golang-migrate for migrations ([Tutorial](https://github.com/golang-migrate/migrate/blob/master/database/postgres/TUTORIAL.md))
and sqlc for queries, mutations & codegen ([Tutorial](https://docs.sqlc.dev/en/stable/tutorials/getting-started-postgresql.html)).

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

- call LLM with a user's query + related snippets from the book (implement RAG endpoint)

## Further Improvements

- ollama as a component in docker-compose with necessary embedding model
  pre-installed. So we don't have to require manually installing it.
- insert embeddings into DB as they are created to prevent large memory spikes
  for larger books
- improve the chunking mechanism
- include mechanism to "expand" a passage to read e.g. what comes before/after
  it
