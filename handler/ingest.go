// Package handler implements all REST API route handlers
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/embiem/book-rag/data"
	"github.com/embiem/book-rag/db"
	"github.com/embiem/book-rag/rag"
	"github.com/pgvector/pgvector-go"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type IngestBookSuccessResponse struct {
	Message    string `json:"message"`
	BookName   string `json:"book_name"`
	BookID     int64  `json:"book_id"`
	TextSize   int    `json:"text_size"`
	ChunkCount int    `json:"chunk_count"`
}

func HandleIngestBook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)

	// Parse multipart form with max memory of 32MB
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(ErrorResponse{Error: "Failed to parse multipart form: " + err.Error()})
		return
	}

	// Get book name from form data
	bookName := r.FormValue("name")
	if bookName == "" {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(ErrorResponse{Error: "Book name is required"})
		return
	}

	var text string

	// Check if text is provided directly in the form
	if directText := r.FormValue("text"); directText != "" {
		text = directText
	} else {
		// Otherwise, try toget text from uploaded .txt file
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(ErrorResponse{Error: "Either 'text' or 'file' must be provided"})
			return
		}
		defer file.Close()

		if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".txt") {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(ErrorResponse{Error: "Only .txt files are accepted"})
			return
		}

		fileRaw, err := io.ReadAll(file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			enc.Encode(ErrorResponse{Error: "Failed to read file content: " + err.Error()})
			return
		}

		text = string(fileRaw)
	}

	if strings.TrimSpace(text) == "" {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(ErrorResponse{Error: "Text content cannot be empty"})
		return
	}

	tx, err := db.Conn.Begin(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}
	defer tx.Rollback(r.Context())

	// All good, now we can insert to db, chunk & create embeddings
	qtx := db.Queries.WithTx(tx)
	book, err := qtx.CreateBook(r.Context(), data.CreateBookParams{
		BookName: bookName,
		BookText: text,
	})
	if err != nil {
		slog.Error("Could not create book in DB", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}

	// Split text into chunks
	chunks := rag.ChunkText(text)
	if len(chunks) == 0 {
		slog.Error("No chunks generated from text")
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}

	slog.Info("Generated text chunks", "count", len(chunks), "book_id", book.ID)

	// Generate embeddings for all chunks
	embeddings, err := rag.GenerateEmbeddings(chunks)
	if err != nil {
		slog.Error("Failed to generate embeddings", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}

	if len(embeddings) != len(chunks) {
		slog.Error("Embeddings count mismatch", "embeddings", len(embeddings), "chunks", len(chunks))
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}

	// Prepare batch insert parameters
	passageParams := make([]data.CreateBookPassagesParams, len(chunks))
	for i, chunk := range chunks {
		passageParams[i] = data.CreateBookPassagesParams{
			BookID:      book.ID,
			PassageText: chunk,
			Embedding:   pgvector.NewVector(embeddings[i]),
		}
	}

	// Batch insert all passages with embeddings
	batchResults := qtx.CreateBookPassages(r.Context(), passageParams)
	var batchErr error
	batchResults.Exec(func(i int, err error) {
		if err != nil {
			slog.Error("Failed to insert passage", "index", i, "err", err)
			batchErr = err
		}
	})

	if batchErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ErrorResponse{Error: "Failed to save passages to database"})
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		slog.Error("Failed to commit transaction", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ErrorResponse{Error: "Internal Server Error"})
		return
	}

	fmt.Println("=== Book Ingestion ===")
	fmt.Printf("Book ID: %d\n", book.ID)
	fmt.Printf("Book Name: %s\n", bookName)
	fmt.Printf("Text Size: %d characters\n", len(text))
	fmt.Printf("Chunks Created: %d\n", len(chunks))
	// fmt.Println("Text Content:")
	// fmt.Println(text)
	fmt.Println("======================")

	// Return success response
	w.WriteHeader(http.StatusOK)
	enc.Encode(IngestBookSuccessResponse{
		Message:    "Book ingested successfully with embeddings",
		BookName:   bookName,
		BookID:     book.ID,
		TextSize:   len(text),
		ChunkCount: len(chunks),
	})
}
