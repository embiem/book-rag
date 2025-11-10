package rag

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateEmbedding(t *testing.T) {
	mockEmbedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	mockResponse := OllamaResponseEmbeddings{
		Embedding: mockEmbedding,
	}

	testInputText := "test input text"

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify the request path
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("Expected path /api/embeddings, got %s", r.URL.Path)
		}

		// Verify Content-Type header
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify the request body
		var payload OllamaPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if payload.Input != testInputText {
			t.Errorf("Expected input %s, got %s", testInputText, payload.Input)
		}

		// Write the mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	originalURL := OllamaBaseURL
	OllamaBaseURL = server.URL
	defer func() { OllamaBaseURL = originalURL }()

	embedding, err := GenerateEmbedding(testInputText)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(embedding) != len(mockEmbedding) {
		t.Fatalf("Expected embedding length %d, got %d", len(mockEmbedding), len(embedding))
	}

	for i, val := range embedding {
		if val != mockEmbedding[i] {
			t.Errorf("Expected embedding[%d] = %f, got %f", i, mockEmbedding[i], val)
		}
	}
}

func TestGenerateEmbedding_ServerError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	originalURL := OllamaBaseURL
	OllamaBaseURL = server.URL
	defer func() { OllamaBaseURL = originalURL }()

	_, err := GenerateEmbedding("test input text")

	if err == nil {
		t.Fatal("Expected an error, got nil")
	}
}

func TestGenerateEmbedding_InvalidJSON(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	originalURL := OllamaBaseURL
	OllamaBaseURL = server.URL
	defer func() { OllamaBaseURL = originalURL }()

	_, err := GenerateEmbedding("test input text")

	if err == nil {
		t.Fatal("Expected an error, got nil")
	}
}
