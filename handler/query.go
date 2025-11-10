package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/embiem/book-rag/data"
	"github.com/embiem/book-rag/db"
	"github.com/embiem/book-rag/rag"
	"github.com/go-chi/chi/v5"
	"github.com/pgvector/pgvector-go"
)

type QueryBookRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type QueryBookResponse struct {
	BookID  int64           `json:"book_id"`
	Query   string          `json:"query"`
	Limit   int32           `json:"limit"`
	Results []PassageResult `json:"results"`
}

type PassageResult struct {
	ID         int64   `json:"id"`
	Text       string  `json:"text"`
	Similarity float32 `json:"similarity"`
}

func HandleQueryBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)

	decoder := json.NewDecoder(r.Body)
	var payload QueryBookRequest
	err := decoder.Decode(&payload)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(ErrorResponse{Error: "Invalid Request params"})
	}

	bookIDStr := chi.URLParam(r, "bookID")
	bookID, err := strconv.ParseInt(bookIDStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(ErrorResponse{Error: "Invalid or missing book ID"})
		return
	}

	exists, err := db.Queries.BookExists(r.Context(), bookID)
	if err != nil {
		slog.Error("Failed to check if book exists", "err", err, "book_id", bookID)
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		enc.Encode(ErrorResponse{Error: "Book not found"})
		return
	}

	if payload.Query == "" {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(ErrorResponse{Error: "Query parameter is required"})
		return
	}

	// Optional limit parameter (default 20, max 100)
	limit := int32(20)
	if payload.Limit > 0 {
		limit = min(int32(payload.Limit), 100)
	}

	embeddings, err := rag.GenerateEmbeddings([]string{payload.Query})
	if err != nil {
		slog.Error("Failed to generate embedding for query", "err", err, "query", payload.Query)
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}

	if len(embeddings) == 0 {
		slog.Error("No embeddings returned for query", "query", payload.Query)
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}

	queryEmbedding := pgvector.NewVector(embeddings[0])

	results, err := db.Queries.QueryBook(r.Context(), data.QueryBookParams{
		BookID:    bookID,
		Embedding: queryEmbedding,
		Limit:     limit,
	})
	if err != nil {
		slog.Error("Failed to query book passages", "err", err, "book_id", bookID)
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}

	passages := make([]PassageResult, len(results))
	for i, result := range results {
		passages[i] = PassageResult{
			ID:         result.ID,
			Text:       result.PassageText,
			Similarity: result.Similarity,
		}
	}

	w.WriteHeader(http.StatusOK)
	enc.Encode(QueryBookResponse{
		BookID:  bookID,
		Query:   payload.Query,
		Limit:   limit,
		Results: passages,
	})
}
