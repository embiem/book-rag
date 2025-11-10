// Package rag implements vector embeddings & llm gen
package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

func GenerateEmbeddings(input []string) ([][]float32, error) {
	payload := OllamaPayload{
		Model: EmbeddingModel,
		Input: input,
	}

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

	if len(data.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama API returned empty embeddings array")
	}

	return data.Embeddings, nil
}
