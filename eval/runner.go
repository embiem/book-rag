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
func RunEvaluation(
	ctx context.Context, client *openai.Client, dataset *EvalDataset, ragBaseURL string,
) (*EvalRun, error) {
	run := &EvalRun{
		Version: "1.0",
		RunAt:   time.Now(),
		Results: make([]EvalResult, 0, len(dataset.QAPairs)),
	}

	fmt.Printf("Running evaluation on %d QA pairs...\n", len(dataset.QAPairs))

	for i, qa := range dataset.QAPairs {
		fmt.Printf("Evaluating %d/%d: %s\n", i+1, len(dataset.QAPairs), qa.Question)

		// Query the RAG system
		generatedAnswer, numChunks, retrievedContext, err := queryRAGSystem(ctx, ragBaseURL, qa.BookID, qa.Question)
		if err != nil {
			fmt.Printf("  Warning: Failed to query RAG system: %v\n", err)
			// Record failed result (not included in scoring)
			run.Results = append(run.Results, EvalResult{
				QAID:            qa.ID,
				Question:        qa.Question,
				ReferenceAnswer: qa.ReferenceAnswer,
				GeneratedAnswer: fmt.Sprintf("ERROR: %v", err),
				RetrievedChunks: 0,
				Scores: RAGEvalScores{
					Reasoning: "System error - could not generate answer",
				},
				EvaluatedAt: time.Now(),
				Failed:      true,
			})
			continue
		}

		// Judge the answer across multiple dimensions with retry logic
		fmt.Printf("  Evaluating answer across 4 dimensions (faithfulness, relevance, correctness, context)...\n")

		var scores RAGEvalScores
		var judgeErr error
		failed := false
		maxRetries := 3

		for attempt := 1; attempt <= maxRetries; attempt++ {
			scores, judgeErr = JudgeAnswer(ctx, client, qa.Question, qa.ReferenceAnswer, generatedAnswer, retrievedContext)
			if judgeErr == nil {
				// Success!
				break
			}

			if attempt < maxRetries {
				fmt.Printf("  Warning: Evaluation attempt %d/%d failed: %v\n", attempt, maxRetries, judgeErr)
				fmt.Printf("  Retrying...\n")
			} else {
				// Final attempt failed
				fmt.Printf("  Error: All %d evaluation attempts failed: %v\n", maxRetries, judgeErr)
				scores = RAGEvalScores{
					Reasoning: fmt.Sprintf("Evaluation error after %d attempts: %v", maxRetries, judgeErr),
				}
				failed = true
			}
		}

		result := EvalResult{
			QAID:             qa.ID,
			Question:         qa.Question,
			ReferenceAnswer:  qa.ReferenceAnswer,
			GeneratedAnswer:  generatedAnswer,
			RetrievedChunks:  numChunks,
			RetrievedContext: retrievedContext,
			Scores:           scores,
			EvaluatedAt:      time.Now(),
			Failed:           failed,
		}

		run.Results = append(run.Results, result)
		if !failed {
			fmt.Printf("  Scores - Faithfulness: %d, Relevance: %d, Correctness: %d, Context: %d\n",
				scores.Faithfulness, scores.AnswerRelevance, scores.Correctness, scores.ContextRelevance)
		}
	}

	// Calculate metrics
	fmt.Printf("\nCalculating metrics...\n")
	run.Metrics = CalculateMetrics(run.Results)

	return run, nil
}

// queryRAGSystem sends a query to the RAG system and returns the generated answer and retrieved context
func queryRAGSystem(ctx context.Context, baseURL string, bookID int64, query string) (string, int, string, error) {
	url := fmt.Sprintf("%s/books/%d/rag", baseURL, bookID)

	requestBody := map[string]string{
		"query": query,
	}
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request with context for cancellation support
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, "", fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Answer           string `json:"answer"`
		RetrievedChunks  int    `json:"retrieved_chunks,omitempty"`
		RetrievedContext string `json:"retrieved_context,omitempty"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", 0, "", fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Answer, response.RetrievedChunks, response.RetrievedContext, nil
}
