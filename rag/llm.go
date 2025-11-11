package rag

import (
	"context"
	"log/slog"

	"github.com/openai/openai-go/v3"
)

func GenerateText(ctx context.Context, input string) (response string, err error) {
	client := openai.NewClient() // OPENAI_API_KEY env var should be available

	slog.Info("Calling OpenAI...", "prompt", input, "model", openai.ChatModelGPT5Mini)

	chatCompletion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(input),
		},
		Model: openai.ChatModelGPT5Mini,
	})
	if err != nil {
		return "", err
	}
	return chatCompletion.Choices[0].Message.Content, nil
}
