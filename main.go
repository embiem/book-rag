package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/embiem/book-rag/db"
	"github.com/embiem/book-rag/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	initialize()
	defer teardown()

	r := chi.NewRouter()
	r.Use(middleware.CleanPath)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(middleware.Logger)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`Book RAG API

Available endpoints:
- GET /books - List available books for querying
- POST /books - Ingest a new book into the vector database (upload .txt file)
- POST /books/{bookID}/query - Query for snippets from a specific book
  Body: {"query": "search text", "limit": 20}
  query (required), limit (optional, default: 20, max: 100)
`))
	})

	r.Get("/books", handler.HandleListBooks)

	r.Post("/books", handler.HandleIngestBook)

	r.Post("/books/{bookID}/query", handler.HandleQueryBook)

	r.Post("/books/{bookID}/rag", handler.HandleGenerate)

	slog.Info("Listening on :3000")
	http.ListenAndServe(":3000", r)
}

func initialize() {
	openaiApiKey := os.Getenv("OPENAI_API_KEY")
	if openaiApiKey == "" {
		slog.Warn("Missing OPENAI_API_KEY env var. /rag endpoint won't work.")
	}

	// Setup DB
	if err := db.Init(); err != nil {
		log.Fatalf("couldn't init db: %v", err)
	}
}

func teardown() {
	slog.Info("Teardown started...")
	db.Teardown()
	slog.Info("Teardown finished.")
}
