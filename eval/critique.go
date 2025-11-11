package eval

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
)

// CritiqueQAPair evaluates a QA pair on three quality dimensions in parallel
func CritiqueQAPair(ctx context.Context, client *openai.Client, qa QAPair) (CritiqueScores, error) {
	scores := CritiqueScores{}

	// Run all three critiques in parallel
	type result struct {
		score     int
		reasoning string
		err       error
	}

	groundCh := make(chan result, 1)
	relCh := make(chan result, 1)
	standCh := make(chan result, 1)

	// Groundedness critique
	go func() {
		score, reason, err := critiqueGroundedness(ctx, client, qa)
		groundCh <- result{score, reason, err}
	}()

	// Relevance critique
	go func() {
		score, reason, err := critiqueRelevance(ctx, client, qa)
		relCh <- result{score, reason, err}
	}()

	// Standalone critique
	go func() {
		score, reason, err := critiqueStandalone(ctx, client, qa)
		standCh <- result{score, reason, err}
	}()

	// Collect results
	groundResult := <-groundCh
	relResult := <-relCh
	standResult := <-standCh

	// Check for errors
	if groundResult.err != nil {
		return scores, fmt.Errorf("groundedness critique failed: %w", groundResult.err)
	}
	if relResult.err != nil {
		return scores, fmt.Errorf("relevance critique failed: %w", relResult.err)
	}
	if standResult.err != nil {
		return scores, fmt.Errorf("standalone critique failed: %w", standResult.err)
	}

	scores.Groundedness = groundResult.score
	scores.Relevance = relResult.score
	scores.Standalone = standResult.score
	scores.Reasoning = fmt.Sprintf("Groundedness: %s\nRelevance: %s\nStandalone: %s",
		groundResult.reasoning, relResult.reasoning, standResult.reasoning)

	return scores, nil
}

// critiqueGroundedness checks if the question can be answered from the context
func critiqueGroundedness(ctx context.Context, client *openai.Client, qa QAPair) (int, string, error) {
	prompt := fmt.Sprintf(`Evaluate if the following question can be fully answered using ONLY the provided context.

Context:
%s

Question: %s
Reference Answer: %s

Score 1-5:
1: Cannot be answered at all from context
2: Requires significant external knowledge
3: Partially answerable from context
4: Mostly answerable from context
5: Fully answerable from context alone

First provide your reasoning, then output "Score: X" where X is 1-5.`, qa.Context, qa.Question, qa.ReferenceAnswer)

	return callLLMForScore(ctx, client, prompt)
}

// critiqueRelevance checks if the question is useful to real users
func critiqueRelevance(ctx context.Context, client *openai.Client, qa QAPair) (int, string, error) {
	prompt := fmt.Sprintf(`Evaluate if this question would be useful to someone reading or researching a book.

Question: %s
Reference Answer: %s

Score 1-5:
1: Trivial or useless question
2: Slightly useful but very basic
3: Moderately useful
4: Quite useful and insightful
5: Highly valuable question for understanding the book

First provide your reasoning, then output "Score: X" where X is 1-5.`, qa.Question, qa.ReferenceAnswer)

	return callLLMForScore(ctx, client, prompt)
}

// critiqueStandalone checks if the question is clear without additional context
func critiqueStandalone(ctx context.Context, client *openai.Client, qa QAPair) (int, string, error) {
	prompt := fmt.Sprintf(`Evaluate if this question is understandable and well-formed on its own, without needing the source context.

Question: %s

Score 1-5:
1: Unintelligible or requires context to understand
2: Mostly unclear without context
3: Somewhat understandable but could be clearer
4: Clear and understandable
5: Perfectly clear and well-formed question

First provide your reasoning, then output "Score: X" where X is 1-5.`, qa.Question)

	return callLLMForScore(ctx, client, prompt)
}

// callLLMForScore makes an LLM call and extracts a 1-5 score
func callLLMForScore(ctx context.Context, client *openai.Client, prompt string) (int, string, error) {
	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4oMini,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return 0, "", fmt.Errorf("openai api call failed: %w", err)
	}

	if len(completion.Choices) == 0 {
		return 0, "", fmt.Errorf("no response from openai")
	}

	response := completion.Choices[0].Message.Content

	// Parse score and reasoning using shared helper
	return parseScoreFromResponse(response)
}
