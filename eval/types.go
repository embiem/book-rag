// Package eval implements a RAG evaluation pipeline
package eval

import "time"

// QAPair represents a question-answer pair with context and quality scores
type QAPair struct {
	ID              string         `json:"id"`
	Question        string         `json:"question"`
	ReferenceAnswer string         `json:"reference_answer"`
	Context         string         `json:"context"` // The chunk used to generate this QA
	BookID          int64          `json:"book_id"`
	CritiqueScores  CritiqueScores `json:"critique_scores"`
	GeneratedAt     time.Time      `json:"generated_at"`
}

// CritiqueScores holds the three quality dimensions for filtering
type CritiqueScores struct {
	Groundedness int    `json:"groundedness"` // 1-5: Can question be answered from context?
	Relevance    int    `json:"relevance"`    // 1-5: Is question useful to users?
	Standalone   int    `json:"standalone"`   // 1-5: Is question clear without context?
	Reasoning    string `json:"reasoning"`    // Combined reasoning from critiques
}

// PassesQualityFilter checks if QA pair meets minimum quality threshold (≥3 on all)
func (c CritiqueScores) PassesQualityFilter() bool {
	return c.Groundedness >= 3 && c.Relevance >= 3 && c.Standalone >= 3
}

// RAGEvalScores holds multi-dimensional RAG evaluation scores (RAGAS-style)
type RAGEvalScores struct {
	Faithfulness     int    `json:"faithfulness"`      // 1-5: Is answer grounded in retrieved context only?
	AnswerRelevance  int    `json:"answer_relevance"`  // 1-5: Does answer address the question?
	Correctness      int    `json:"correctness"`       // 1-5: Is answer factually accurate vs reference?
	ContextRelevance int    `json:"context_relevance"` // 1-5: Is retrieved context relevant to question?
	Reasoning        string `json:"reasoning"`         // Combined reasoning from all dimensions
}

// EvalDataset represents a collection of QA pairs for evaluation
type EvalDataset struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	QAPairs   []QAPair  `json:"qa_pairs"`
}

// EvalResult represents the result of evaluating one QA pair
type EvalResult struct {
	QAID            string        `json:"qa_id"`
	Question        string        `json:"question"`
	ReferenceAnswer string        `json:"reference_answer"`
	GeneratedAnswer string        `json:"generated_answer"`
	RetrievedChunks int           `json:"retrieved_chunks"` // Number of passages retrieved
	RetrievedContext string       `json:"retrieved_context,omitempty"` // Context passages for faithfulness check
	Scores          RAGEvalScores `json:"scores"`           // Multi-dimensional evaluation scores
	EvaluatedAt     time.Time     `json:"evaluated_at"`
	Failed          bool          `json:"failed,omitempty"` // True if RAG system or judging failed
}

// EvalRun represents a complete evaluation run with all results
type EvalRun struct {
	Version     string       `json:"version"`
	DatasetFile string       `json:"dataset_file"`
	RunAt       time.Time    `json:"run_at"`
	Results     []EvalResult `json:"results"`
	Metrics     EvalMetrics  `json:"metrics"`
}

// DimensionMetrics holds metrics for a single evaluation dimension
type DimensionMetrics struct {
	AverageScore        float64         `json:"average_score"`
	MedianScore         float64         `json:"median_score"`
	ScoreDistribution   map[int]int     `json:"score_distribution"`    // Count of each score 1-5
	AccuracyAtThreshold map[int]float64 `json:"accuracy_at_threshold"` // % scoring >= threshold
	PassRateAtFour      float64         `json:"pass_rate"`             // Percentage scoring >= 4
}

// EvalMetrics holds aggregate statistics from an evaluation run
type EvalMetrics struct {
	TotalQuestions   int              `json:"total_questions"`
	FailedQuestions  int              `json:"failed_questions,omitempty"` // Questions that failed (RAG or judging errors)
	Faithfulness     DimensionMetrics `json:"faithfulness"`
	AnswerRelevance  DimensionMetrics `json:"answer_relevance"`
	Correctness      DimensionMetrics `json:"correctness"`
	ContextRelevance DimensionMetrics `json:"context_relevance"`
}
