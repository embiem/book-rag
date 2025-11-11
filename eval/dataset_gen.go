package eval

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/openai/openai-go/v3"
)

// GenerateQAPair creates a question-answer pair from a text chunk using LLM
func GenerateQAPair(ctx context.Context, client *openai.Client, context string, bookID int64) (QAPair, error) {
	prompt := fmt.Sprintf(`Based on the following text from a book, generate ONE factoid question that can be answered using only this text.
Then provide a concise answer to that question.

Text:
%s

Generate a question that:
- Can be fully answered from the text above
- Would be useful to someone studying or understanding this book
- Is COMPLETELY CLEAR and understandable on its own without needing the source text
- Uses FULL NAMES instead of pronouns (don't use "he", "she", "they" - use actual character names)
- Is specific and focuses on facts, events, or concepts mentioned in the text

IMPORTANT: The question must be perfectly understandable by someone who hasn't read the text above.

Output format:
Question: [your question]
Answer: [your answer]`, context)

	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return QAPair{}, fmt.Errorf("openai api call failed: %w", err)
	}

	if len(completion.Choices) == 0 {
		return QAPair{}, fmt.Errorf("no response from openai")
	}

	response := completion.Choices[0].Message.Content

	// Parse question and answer
	question, answer, err := parseQAPairResponse(response)
	if err != nil {
		return QAPair{}, fmt.Errorf("failed to parse QA response: %w", err)
	}

	return QAPair{
		ID:              uuid.New().String(),
		Question:        question,
		ReferenceAnswer: answer,
		Context:         context,
		BookID:          bookID,
		GeneratedAt:     time.Now(),
	}, nil
}

// parseQAPairResponse extracts question and answer from LLM response
func parseQAPairResponse(response string) (string, string, error) {
	lines := make([]string, 0)
	for _, line := range splitLines(response) {
		trimmed := trimSpace(line)
		if trimmed != "" {
			lines = append(lines, trimmed)
		}
	}

	var question, answer string
	for _, line := range lines {
		if len(line) > 10 && (startsWith(line, "Question:") || startsWith(line, "Q:")) {
			question = extractAfterPrefix(line, []string{"Question:", "Q:"})
		} else if len(line) > 8 && (startsWith(line, "Answer:") || startsWith(line, "A:")) {
			answer = extractAfterPrefix(line, []string{"Answer:", "A:"})
		}
	}

	if question == "" || answer == "" {
		return "", "", fmt.Errorf("could not parse question and answer from response: %s", response)
	}

	return question, answer, nil
}

// GenerateDataset creates a complete evaluation dataset
func GenerateDataset(ctx context.Context, client *openai.Client, chunks []ChunkWithBook, targetSamples int) (*EvalDataset, error) {
	dataset := &EvalDataset{
		Version:   "1.0",
		CreatedAt: time.Now(),
		QAPairs:   make([]QAPair, 0),
	}

	generated := 0
	filtered := 0

	fmt.Printf("Generating %d QA pairs (expecting ~%d after filtering)...\n", targetSamples, targetSamples*2/5)

	// Shuffle chunks to get diverse samples
	shuffled := make([]ChunkWithBook, len(chunks))
	copy(shuffled, chunks)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	for i := 0; i < len(shuffled) && generated < targetSamples; i++ {
		chunk := shuffled[i]

		fmt.Printf("Generating QA pair %d/%d...\n", generated+1, targetSamples)

		// Generate QA pair
		qa, err := GenerateQAPair(ctx, client, chunk.Text, chunk.BookID)
		if err != nil {
			fmt.Printf("  Warning: Failed to generate QA pair: %v\n", err)
			continue
		}
		generated++

		// Critique the QA pair
		fmt.Printf("  Critiquing QA pair...\n")
		scores, err := CritiqueQAPair(ctx, client, qa)
		if err != nil {
			fmt.Printf("  Warning: Failed to critique QA pair: %v\n", err)
			continue
		}

		qa.CritiqueScores = scores

		// Filter based on quality
		if scores.PassesQualityFilter() {
			dataset.QAPairs = append(dataset.QAPairs, qa)
			filtered++
			fmt.Printf("  ✓ Accepted (scores: G=%d, R=%d, S=%d) - Total: %d\n",
				scores.Groundedness, scores.Relevance, scores.Standalone, filtered)
		} else {
			fmt.Printf("  ✗ Rejected (scores: G=%d, R=%d, S=%d)\n",
				scores.Groundedness, scores.Relevance, scores.Standalone)
		}
	}

	// Calculate percentage, avoiding division by zero
	var percentage float64
	if generated > 0 {
		percentage = float64(filtered) / float64(generated) * 100
	}

	fmt.Printf("\nDataset generation complete: %d generated, %d passed filtering (%.1f%%)\n",
		generated, filtered, percentage)

	return dataset, nil
}

// ChunkWithBook represents a text chunk with its book ID
type ChunkWithBook struct {
	Text   string
	BookID int64
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r' || s[start] == '\n') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func extractAfterPrefix(s string, prefixes []string) string {
	for _, prefix := range prefixes {
		if startsWith(s, prefix) {
			result := s[len(prefix):]
			return trimSpace(result)
		}
	}
	return s
}
