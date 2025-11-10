// Package rag implements vector embeddings & llm gen
package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"time"
)

type OllamaPayload struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type OllamaResponseEmbeddings struct {
	Embeddings [][]float32 `json:"embeddings"`
}

const (
	EmbeddingModel string = "embeddinggemma"
)

var OllamaBaseURL string = os.Getenv("OLLAMA_BASE_URL")

const BatchSize = 20

func callOllama(payload OllamaPayload) (*OllamaResponseEmbeddings, error) {
	reqData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/embed", OllamaBaseURL), bytes.NewReader(reqData))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	httpClient := http.Client{Timeout: 30 * time.Second}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	resData, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama API returned status %d: %s", res.StatusCode, string(resData))
	}

	var data OllamaResponseEmbeddings
	err = json.Unmarshal(resData, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func GenerateEmbeddings(input []string) ([][]float32, error) {
	// Create embeddings in batches, one at a time to prevent overwhelming
	allEmbeddings := make([][]float32, 0, len(input))

	batchCount := math.Ceil(float64(len(input)) / float64(BatchSize))
	for i := 0; i < int(batchCount); i++ {
		slog.Info("Generating Embeddings...", "batch", i+1, "of", batchCount)
		startIdx := i * BatchSize
		batch := input[startIdx:min(startIdx+BatchSize, len(input))]

		payload := OllamaPayload{
			Model: EmbeddingModel,
			Input: batch,
		}

		res, err := callOllama(payload)
		if err != nil {
			return nil, err
		}

		allEmbeddings = append(allEmbeddings, res.Embeddings...)
	}

	if len(allEmbeddings) == 0 {
		return nil, fmt.Errorf("ollama API returned empty embeddings array")
	}

	return allEmbeddings, nil
}
