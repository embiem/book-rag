package eval

import (
	"context"
	"fmt"

	"github.com/openai/openai-go/v3"
)

// JudgeAnswer evaluates a RAG-generated answer across multiple dimensions (RAGAS-style)
// It evaluates faithfulness, answer relevance, correctness, and context relevance in parallel
func JudgeAnswer(ctx context.Context, client *openai.Client, question, reference, generated, retrievedContext string) (RAGEvalScores, error) {
	scores := RAGEvalScores{}

	// Run all four evaluations in parallel
	type result struct {
		score     int
		reasoning string
		err       error
	}

	faithCh := make(chan result, 1)
	ansRelCh := make(chan result, 1)
	correctCh := make(chan result, 1)
	ctxRelCh := make(chan result, 1)

	// Faithfulness/Groundedness - is answer based only on retrieved context?
	go func() {
		score, reason, err := judgeFaithfulness(ctx, client, question, generated, retrievedContext)
		faithCh <- result{score, reason, err}
	}()

	// Answer Relevance - does answer address the question?
	go func() {
		score, reason, err := judgeAnswerRelevance(ctx, client, question, generated)
		ansRelCh <- result{score, reason, err}
	}()

	// Correctness - is answer factually accurate vs reference?
	go func() {
		score, reason, err := judgeCorrectness(ctx, client, question, reference, generated)
		correctCh <- result{score, reason, err}
	}()

	// Context Relevance - is retrieved context relevant to question?
	go func() {
		score, reason, err := judgeContextRelevance(ctx, client, question, retrievedContext)
		ctxRelCh <- result{score, reason, err}
	}()

	// Collect results
	faithResult := <-faithCh
	ansRelResult := <-ansRelCh
	correctResult := <-correctCh
	ctxRelResult := <-ctxRelCh

	// Check for errors
	if faithResult.err != nil {
		return scores, fmt.Errorf("faithfulness evaluation failed: %w", faithResult.err)
	}
	if ansRelResult.err != nil {
		return scores, fmt.Errorf("answer relevance evaluation failed: %w", ansRelResult.err)
	}
	if correctResult.err != nil {
		return scores, fmt.Errorf("correctness evaluation failed: %w", correctResult.err)
	}
	if ctxRelResult.err != nil {
		return scores, fmt.Errorf("context relevance evaluation failed: %w", ctxRelResult.err)
	}

	scores.Faithfulness = faithResult.score
	scores.AnswerRelevance = ansRelResult.score
	scores.Correctness = correctResult.score
	scores.ContextRelevance = ctxRelResult.score
	scores.Reasoning = fmt.Sprintf("Faithfulness: %s\n\nAnswer Relevance: %s\n\nCorrectness: %s\n\nContext Relevance: %s",
		faithResult.reasoning, ansRelResult.reasoning, correctResult.reasoning, ctxRelResult.reasoning)

	return scores, nil
}

// judgeFaithfulness checks if the answer is grounded only in the retrieved context
// This is the anti-hallucination metric - critical for RAG systems
func judgeFaithfulness(ctx context.Context, client *openai.Client, question, answer, context string) (int, string, error) {
	prompt := fmt.Sprintf(`Evaluate if the generated answer is fully grounded in the provided context. All claims in the answer must be verifiable from the context alone.

Question: %s

Retrieved Context:
%s

Generated Answer: %s

Score 1-5:
1: Answer contains mostly hallucinated information not in context
2: Answer contains significant information not found in context
3: Answer is partially grounded but includes some unverified claims
4: Answer is mostly grounded with minor unsupported details
5: Answer is fully grounded - all claims can be verified from context

First provide your detailed reasoning, then output "Score: X" where X is 1-5.`, question, context, answer)

	return callLLMForScore(ctx, client, prompt)
}

// judgeAnswerRelevance checks if the answer actually addresses the question asked
func judgeAnswerRelevance(ctx context.Context, client *openai.Client, question, answer string) (int, string, error) {
	prompt := fmt.Sprintf(`Evaluate how well the generated answer addresses the specific question asked.

Question: %s

Generated Answer: %s

Score 1-5:
1: Answer is completely irrelevant to the question
2: Answer is mostly off-topic or addresses wrong question
3: Answer is partially relevant but misses key aspects
4: Answer is mostly relevant with minor tangents
5: Answer directly and fully addresses the question

First provide your detailed reasoning, then output "Score: X" where X is 1-5.`, question, answer)

	return callLLMForScore(ctx, client, prompt)
}

// judgeCorrectness evaluates factual accuracy compared to the reference answer
func judgeCorrectness(ctx context.Context, client *openai.Client, question, reference, generated string) (int, string, error) {
	prompt := fmt.Sprintf(`Evaluate how correct the generated answer is compared to the reference answer. Consider:
- Factual accuracy
- Completeness of information
- Alignment with the reference

Question: %s

Reference Answer: %s

Generated Answer: %s

Score 1-5:
1: Completely incorrect - contradicts reference or provides wrong information
2: Mostly incorrect - contains some truth but major errors or omissions
3: Partially correct - has relevant information but missing key details
4: Mostly correct - captures main points with minor omissions
5: Fully correct - accurately and completely answers the question

First provide your detailed reasoning, then output "Score: X" where X is 1-5.`, question, reference, generated)

	return callLLMForScore(ctx, client, prompt)
}

// judgeContextRelevance evaluates if the retrieved context is relevant to the question
// This measures retrieval quality
func judgeContextRelevance(ctx context.Context, client *openai.Client, question, context string) (int, string, error) {
	prompt := fmt.Sprintf(`Evaluate how relevant the retrieved context is to answering the question. This measures retrieval quality.

Question: %s

Retrieved Context:
%s

Score 1-5:
1: Context is completely irrelevant to the question
2: Context has minimal relevance to the question
3: Context is somewhat relevant but missing key information
4: Context is mostly relevant with good information
5: Context is highly relevant and contains all needed information

First provide your detailed reasoning, then output "Score: X" where X is 1-5.`, question, context)

	return callLLMForScore(ctx, client, prompt)
}

// callLLMForScore is defined in critique.go and shared here
