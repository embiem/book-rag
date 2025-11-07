package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
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

	r.Post("/books", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("POST books to ingest a new book into vector db"))
	})

	r.Get("/books/{bookID}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GET book with ID to query for snippets from the given book"))
	})

	http.ListenAndServe(":3000", r)
}
