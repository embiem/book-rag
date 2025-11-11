# Book RAG

This is a HTTP server in Golang that offers REST endpoints for ingesting a book
into a vector database and then querying it and doing RAG using a LLM on it.

The evaluation pipeline was developed utilizing Claude Code.
It's based on a common RAG evaluation concept of LLM-as-a-judge.

## Prerequisites

- mise-en-place: <https://mise.jdx.dev/>
- docker

## Get started

1. activate mise in the project (pins a go version & loads env vars)
2. `docker compose up -d` to run the necessary architecture (like db)
3. install [ollama](https://ollama.com/download) (recommended: latest. minimum: v0.11.10)
   & run `ollama pull embeddinggemma`
   to download the vector embeddings model used for generating vector embeddings
4. ensure the `OPENAI_API_KEY` env var exists and has a valid OpenAI API Key for
   the LLM generation, for example using mise.local.toml
5. run `air` for hot reloaded development or `go run main.go`

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
  -F "name=Romeo and Juliet" \
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

## Evaluation Pipeline

The project includes a evaluation system to measure and improve RAG
performance using LLM-as-a-judge methodology.

### Overview

The evaluation pipeline consists of:

1. **Synthetic Dataset Generation**: Automatically creates question-answer pairs
   from book passages using a LLM
2. **Quality Filtering**: Three critique agents evaluate each QA pair on:
   - **Groundedness** (1-5): Can the question be answered from the given context?
   - **Relevance** (1-5): Is the question useful to actual users?
   - **Standalone** (1-5): Is the question understandable without additional context?
3. **Answer Evaluation**: LLM judge compares generated answers against reference answers
4. **Metrics**: Scoring including average correctness

### Commands

#### Generate Evaluation Dataset

Create a synthetic QA dataset from ingested books:

```bash
# Generate from a specific book (recommended)
go run cmd/gendata/main.go -samples 30 -book-id 19

# Generate from all books
go run cmd/gendata/main.go -samples 50

# Custom output path
go run cmd/gendata/main.go -samples 30 -book-id 19 -output testdata/my_dataset.json
```

**Parameters**:

- `-samples`: Number of QA pairs to generate before filtering (default: 250)
- `-book-id`: Specific book ID to use (0 = all books, default: 0)
- `-output`: Output file path (default: `testdata/eval_dataset.json`)

**Quality Filter**: Only QA pairs scoring ≥3 on all three critique dimensions are kept.

#### Run Evaluation

Evaluate the RAG system against a dataset:

```bash
# Run evaluation with defaults
go run cmd/evaluate/main.go

# Specify custom dataset and output
go run cmd/evaluate/main.go \
  -dataset testdata/eval_dataset.json \
  -output testdata/results/experiment_v1.json \
  -rag-url http://localhost:3000
```

**Parameters**:

- `-dataset`: Path to evaluation dataset (default: `testdata/eval_dataset.json`)
- `-output`: Results output path (default: `testdata/results/baseline.json`)
- `-rag-url`: RAG server base URL (default: `http://localhost:3000`)

**Requirements**: Server must be running (`go run main.go`) and `OPENAI_API_KEY` must be set.

### Understanding Results

The evaluation produces detailed metrics:

```
EVALUATION RESULTS
============================================================

Total Questions:     15
Average Score:       4.33 / 5.0
Median Score:        4.0
Pass Rate (≥4):      0.867

Score Distribution:
  5:   7 (46.7%)    # Fully correct
  4:   6 (40.0%)    # Mostly correct
  3:   2 (13.3%)    # Partially correct
  2:   0 (0.0%)     # Mostly incorrect
  1:   0 (0.0%)     # Completely incorrect

Accuracy by Threshold:
  ≥5: 46.7%
  ≥4: 86.7%    # Primary success metric (Pass Rate)
  ≥3: 100.0%
```

**Key Metrics**:

- **Average Score**: Mean correctness (1-5 scale)
- **Pass Rate**: Percentage of answers scoring ≥4 (treat as "correct")
- **Distribution**: Breakdown of all scores for detailed analysis

### Baseline Results

Current baseline (15 QA pairs from Romeo & Juliet):

- **Average Score**: 4.33 / 5.0
- **Pass Rate**: 0.867 (86.7% of answers scored ≥4)
- **Perfect Scores**: 46.7% of answers scored 5/5

### Iterating and Improving

1. **Establish Baseline**: Generate dataset and run initial evaluation
2. **Make Changes**: Modify chunking, retrieval count, prompts, etc.
3. **Re-evaluate**: Run evaluation with same dataset to compare results
4. **Compare**: Use the same dataset file to ensure fair comparison

Example workflow:

```bash
# 1. Generate dataset once
go run cmd/gendata/main.go -samples 30 -book-id 19

# 2. Run baseline evaluation
go run cmd/evaluate/main.go -output testdata/results/baseline.json

# 3. Make improvements to the RAG pipeline
#    (e.g., edit handler/generate.go, rag/chunking.go)

# 4. Re-evaluate with same dataset
go run cmd/evaluate/main.go -output testdata/results/improved_v1.json

# 5. Compare results programmatically or manually
```

The baseline evaluation dataset is fairly minimal at this moment.
For a production system, this dataset should be increased in size and
draw from more than one book, ideally cover many different genres, authors and
eras.

## Further Improvements

**Architectural**:

- Containerize Ollama in docker-compose with necessary embedding model
  pre-installed to eliminate manual setup
- Insert embeddings into DB as they are created to prevent large memory spikes
  for larger books (batch streaming)

**RAG Pipeline**:

- Improve the chunking mechanism to better fit the domain space of books:
  - Detect and preserve chapter boundaries
  - Handle prologues, table of contents separately
- Include contextual/structural info in text passages (page, chapter, entities etc)
- Extract entities from books and add as metadata on passages for hybrid search
  to increase precision of query results
- Implement passage "expansion" mechanism to include before/after context:
  - Add references to previous/next passages in results
  - Allow dynamic context window adjustment
- Query optimization:
  - Let LLM generate an optimized query based on user input (HyDE)
  - Implement query rewriting for better retrieval
- Enhanced generation:
  - Use LLM function calling to allow the model to query for more context
  - Support iterative retrieval during generation
  - Add re-ranking step after initial retrieval

**Evaluation System**:

- Generate larger evaluation datasets for more robust metrics
  - Cover many books from a wide range of genres and authors
- Implement automated comparison tools for A/B testing different configurations
