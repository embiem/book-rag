package main

import (
	"log"
	"log/slog"
	"net/http"

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

	// GET books to see available books for querying
	// POST books to ingest a new book into the vector db
	// GET books/{id} to query a specific book
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome!"))
	})

	r.Get("/books", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GET books, to see available books for querying"))
	})

	r.Post("/books", handler.HandleIngestBook)

	r.Get("/books/{bookID}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GET book with ID to query for snippets from the given book"))
	})

	slog.Info("Listening on :3000")
	http.ListenAndServe(":3000", r)
}

func initialize() {
	// TODO: ensure env vars are available

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
