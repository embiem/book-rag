# Book RAG

This is a HTTP server in Golang that offers REST endpoints for ingesting a book
into a vector database and then querying it and doing RAG using a LLM on it.

## Prerequisites

- mise-en-place: <https://mise.jdx.dev/>
- docker

## Get started

1. activate mise in the project (pins a go version & loads env vars)
2. `docker compose up -d` to run the necessary architecture (like db)
3. install [ollama](https://ollama.com/download) & run `ollama pull embeddinggemma`
   to download the vector embeddings model used for generating vector embeddings
4. ensure the `OPENAI_API_KEY` env var exists and has a valid OpenAI API Key for
   the LLM generation, for example using mise.local.toml

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
  - Request body: `{"query": "What happens in the balcony scene?"}`

#### Example curl commands

```bash
# Ingest a new book
curl -X POST http://localhost:3000/books \
  -F "file=@books/romeo_and_juliet.txt"

# List all books
curl http://localhost:3000/books

# Perform RAG a book (replace {bookID} with actual ID from previous commands)
# Requires OPENAI_API_KEY env variable to be set
curl -X POST http://localhost:3000/books/{bookID}/rag \
  -H "Content-Type: application/json" \
  -d '{"query": "What happens in the balcony scene?"}'

# Query a book (replace {bookID} with actual ID from previous commands)
curl -X POST http://localhost:3000/books/{bookID}/query \
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

## Further Improvements

**Architectural**:

- ollama as a component in docker-compose with necessary embedding model
  pre-installed. So we don't have to require manually installing it.
- insert embeddings into DB as they are created to prevent large memory spikes
  for larger books

**RAG specific**:

- setup evaluation pipeline & metrics before improving the RAG pipeline, e.g.
  using a LLM as a judge approach and a
- improve the chunking mechanism to better fit the domain space of books
  (chapters, prologue, table of contents etc)
- extract entities from book and add as metadata on passages, allowing hybrid
  search to increase precision of query results
- include mechanism to "expand" a passage to read e.g. what comes before/after
  it, by including references to previous/next passages
- let LLM generate an optimized query based on the user's input (HyDE)
- during generation, use LLM function calling to allow the model to query for
  more context if necessary, with an additional similarity search or retrieving
  the previous/next passage of a passage that is of interest
