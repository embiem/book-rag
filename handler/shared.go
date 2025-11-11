package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/embiem/book-rag/db"
	"github.com/go-chi/chi/v5"
)

type HttpError struct {
	msg    string
	status int
}

func (e HttpError) Error() string { return e.msg }

func EnsureBookExists(r *http.Request) (int64, error) {
	bookIDStr := chi.URLParam(r, "bookID")
	bookID, err := strconv.ParseInt(bookIDStr, 10, 64)
	if err != nil {
		return 0, HttpError{msg: "Invalid or missing book ID", status: http.StatusBadRequest}
	}

	exists, err := db.Queries.BookExists(r.Context(), bookID)
	if err != nil {
		slog.Error("Failed to check if book exists", "err", err, "book_id", bookID)
		return 0, HttpError{msg: "Could not check if book exists", status: http.StatusInternalServerError}
	}
	if !exists {
		return 0, HttpError{msg: "Book not found", status: http.StatusInternalServerError}
	}

	return bookID, nil
}
