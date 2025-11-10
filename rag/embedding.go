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
	Model string `json:"model"`
	Input string `json:"input"`
}

type OllamaResponseEmbeddings struct {
	Embedding []float32 `json:"embedding"`
}

const (
	EmbeddingModel string = "embeddinggemma"
)

var OllamaBaseURL string = os.Getenv("OLLAMA_BASE_URL")

func GenerateEmbedding(text string) ([]float32, error) {
	payload := OllamaPayload{
		Model: EmbeddingModel,
		Input: text,
	}

	reqData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/embeddings", OllamaBaseURL), bytes.NewReader(reqData))
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

	var data OllamaResponseEmbeddings
	err = json.Unmarshal(resData, &data)
	if err != nil {
		return nil, err
	}

	return data.Embedding, nil
}

func GenerateEmbeddings(texts []string) ([][]float32, error) {
	// TODO: run multiple in batch. docs: https://docs.ollama.com/capabilities/embeddings#generate-a-batch-of-embeddings
	embeddings := make([][]float32, len(texts))
	for i := range len(texts) {
		embed, err := GenerateEmbedding(texts[i])
		if err != nil {
			return nil, err
		}
		embeddings[i] = embed
	}
	return embeddings, nil
}

func GenerateSingleEmbedding(text string) ([]float32, error) {
	embeddings, err := GenerateEmbeddings([]string{text})
	if err != nil {
		return nil, err
	}

	return embeddings[0], nil
}
