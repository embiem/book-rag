package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/embiem/book-rag/db"
	"github.com/go-chi/chi/v5"
)

type HttpError struct {
	Msg    string
	Status int
}

func (e HttpError) Error() string { return e.Msg }

func EnsureBookExists(r *http.Request) (int64, error) {
	bookIDStr := chi.URLParam(r, "bookID")
	bookID, err := strconv.ParseInt(bookIDStr, 10, 64)
	if err != nil {
		return 0, HttpError{Msg: "Invalid or missing book ID", Status: http.StatusBadRequest}
	}

	exists, err := db.Queries.BookExists(r.Context(), bookID)
	if err != nil {
		slog.Error("Failed to check if book exists", "err", err, "book_id", bookID)
		return 0, HttpError{Msg: "Could not check if book exists", Status: http.StatusInternalServerError}
	}
	if !exists {
		return 0, HttpError{Msg: "Book not found", Status: http.StatusNotFound}
	}

	return bookID, nil
}
