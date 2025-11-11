package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openai/openai-go/v3"
)

// RunEvaluation evaluates a RAG system against a dataset
func RunEvaluation(ctx context.Context, client *openai.Client, dataset *EvalDataset, ragBaseURL string) (*EvalRun, error) {
	run := &EvalRun{
		Version:   "1.0",
		RunAt:     time.Now(),
		Results:   make([]EvalResult, 0, len(dataset.QAPairs)),
	}

	fmt.Printf("Running evaluation on %d QA pairs...\n", len(dataset.QAPairs))

	for i, qa := range dataset.QAPairs {
		fmt.Printf("Evaluating %d/%d: %s\n", i+1, len(dataset.QAPairs), qa.Question)

		// Query the RAG system
		generatedAnswer, numChunks, err := queryRAGSystem(ctx, ragBaseURL, qa.BookID, qa.Question)
		if err != nil {
			fmt.Printf("  Warning: Failed to query RAG system: %v\n", err)
			// Record failed result (not included in scoring)
			run.Results = append(run.Results, EvalResult{
				QAID:             qa.ID,
				Question:         qa.Question,
				ReferenceAnswer:  qa.ReferenceAnswer,
				GeneratedAnswer:  fmt.Sprintf("ERROR: %v", err),
				RetrievedChunks:  0,
				CorrectnessScore: 0, // 0 indicates not scored due to error
				JudgeReasoning:   "System error - could not generate answer",
				EvaluatedAt:      time.Now(),
				Failed:           true,
			})
			continue
		}

		// Judge the answer
		fmt.Printf("  Judging answer...\n")
		score, reasoning, err := JudgeAnswer(ctx, client, qa.Question, qa.ReferenceAnswer, generatedAnswer)
		failed := false
		if err != nil {
			fmt.Printf("  Warning: Failed to judge answer: %v\n", err)
			score = 0 // 0 indicates not scored due to error
			reasoning = fmt.Sprintf("Judging error: %v", err)
			failed = true
		}

		result := EvalResult{
			QAID:             qa.ID,
			Question:         qa.Question,
			ReferenceAnswer:  qa.ReferenceAnswer,
			GeneratedAnswer:  generatedAnswer,
			RetrievedChunks:  numChunks,
			CorrectnessScore: score,
			JudgeReasoning:   reasoning,
			EvaluatedAt:      time.Now(),
			Failed:           failed,
		}

		run.Results = append(run.Results, result)
		fmt.Printf("  Score: %d/5\n", score)
	}

	// Calculate metrics
	fmt.Printf("\nCalculating metrics...\n")
	run.Metrics = CalculateMetrics(run.Results)

	return run, nil
}

// queryRAGSystem sends a query to the RAG system and returns the generated answer
func queryRAGSystem(ctx context.Context, baseURL string, bookID int64, query string) (string, int, error) {
	url := fmt.Sprintf("%s/books/%d/rag", baseURL, bookID)

	requestBody := map[string]string{
		"query": query,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request with context for cancellation support
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read body once for both success and error paths
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Answer          string `json:"answer"`
		RetrievedChunks int    `json:"retrieved_chunks,omitempty"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Answer, response.RetrievedChunks, nil
}
