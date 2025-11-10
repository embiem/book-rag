package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/embiem/book-rag/db"
)

type ListBooksResponse struct {
	Books []BookItem `json:"books"`
}

type BookItem struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func HandleListBooks(w http.ResponseWriter, r *http.Request) {
	books, err := db.Queries.ListBooks(r.Context())
	if err != nil {
		slog.Error("Error retrieving books", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Could not load books"))
		return
	}

	bookItems := make([]BookItem, len(books))
	for i, book := range books {
		bookItems[i] = BookItem{
			ID:   book.ID,
			Name: book.BookName,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ListBooksResponse{
		Books: bookItems,
	}); err != nil {
		slog.Error("Error encoding response", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
