package eval

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
)

// JudgeAnswer evaluates a generated answer against a reference answer
func JudgeAnswer(ctx context.Context, client *openai.Client, question, reference, generated string) (int, string, error) {
	prompt := fmt.Sprintf(`You are evaluating the correctness of a generated answer compared to a reference answer.

Question: %s

Reference Answer: %s

Generated Answer: %s

Evaluate how correct the generated answer is compared to the reference answer. Consider:
- Factual accuracy
- Completeness of information
- Alignment with the reference

Score 1-5:
1: Completely incorrect - contradicts reference or provides wrong information
2: Mostly incorrect - contains some truth but major errors or omissions
3: Partially correct - has relevant information but missing key details
4: Mostly correct - captures main points with minor omissions
5: Fully correct - accurately and completely answers the question

First provide your detailed reasoning, then output "Score: X" where X is 1-5.`, question, reference, generated)

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
