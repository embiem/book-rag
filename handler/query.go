package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"

	"github.com/embiem/book-rag/data"
	"github.com/embiem/book-rag/db"
	"github.com/embiem/book-rag/rag"
	"github.com/pgvector/pgvector-go"
)

type QueryBookRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
}

type QueryBookResponse struct {
	BookID   int64           `json:"book_id"`
	Query    string          `json:"query"`
	Limit    int             `json:"limit"`
	Passages []PassageResult `json:"results"`
}

type PassageResult struct {
	ID         int64   `json:"id"`
	Text       string  `json:"text"`
	Similarity float32 `json:"similarity"`
}

func PrettifyPassages(passageResults []PassageResult) string {
	pretty := ""

	for _, p := range passageResults {
		pretty += fmt.Sprintf("Relevance: %d%%\n", int(math.Round(float64(p.Similarity)*100)))
		pretty += p.Text + "\n\n\n"
	}

	return pretty
}

func QueryBook(ctx context.Context, payload QueryBookRequest, bookID int64) (*QueryBookResponse, error) {
	// Optional limit parameter (default 20, max 100)
	limit := int32(20)
	if payload.Limit > 0 {
		limit = min(int32(payload.Limit), 100)
	}

	embeddings, err := rag.GenerateEmbeddings([]string{payload.Query})
	if err != nil {
		slog.Error("Failed to generate embedding for query", "err", err, "query", payload.Query)
		return nil, err
	}

	if len(embeddings) == 0 {
		slog.Error("No embeddings returned for query", "query", payload.Query)
		return nil, errors.New("No embeddings returned for query")
	}

	queryEmbedding := pgvector.NewVector(embeddings[0])

	results, err := db.Queries.QueryBook(ctx, data.QueryBookParams{
		BookID:    bookID,
		Embedding: queryEmbedding,
		Limit:     limit,
	})
	if err != nil {
		slog.Error("Failed to query book passages", "err", err, "book_id", bookID)
		return nil, err
	}

	passages := make([]PassageResult, len(results))
	for i, result := range results {
		passages[i] = PassageResult{
			ID:         result.ID,
			Text:       result.PassageText,
			Similarity: result.Similarity,
		}
	}

	return &QueryBookResponse{
		BookID:   bookID,
		Query:    payload.Query,
		Limit:    int(limit),
		Passages: passages,
	}, nil
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
		return
	}

	bookID, err := EnsureBookExists(r)
	if err != nil {
		if bookErr, ok := err.(HttpError); ok {
			w.WriteHeader(bookErr.Status)
			enc.Encode(ErrorResponse{Error: bookErr.Msg})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			enc.Encode(ErrorResponse{Error: "Internal Server Error"})
		}
		return
	}

	res, err := QueryBook(r.Context(), payload, bookID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}

	w.WriteHeader(http.StatusOK)
	enc.Encode(res)
}
