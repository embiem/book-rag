package eval

import (
	"sort"
)

// CalculateMetrics computes aggregate statistics from evaluation results across all dimensions
func CalculateMetrics(results []EvalResult) EvalMetrics {
	metrics := EvalMetrics{
		TotalQuestions: len(results),
	}

	if len(results) == 0 {
		return metrics
	}

	// Collect scores from successful results only
	faithScores := make([]int, 0, len(results))
	ansRelScores := make([]int, 0, len(results))
	correctScores := make([]int, 0, len(results))
	ctxRelScores := make([]int, 0, len(results))

	failedCount := 0
	for _, result := range results {
		if result.Failed {
			failedCount++
			continue
		}
		faithScores = append(faithScores, result.Scores.Faithfulness)
		ansRelScores = append(ansRelScores, result.Scores.AnswerRelevance)
		correctScores = append(correctScores, result.Scores.Correctness)
		ctxRelScores = append(ctxRelScores, result.Scores.ContextRelevance)
	}

	metrics.FailedQuestions = failedCount

	// If all results failed, return empty metrics
	if len(faithScores) == 0 {
		return metrics
	}

	// Calculate metrics for each dimension
	metrics.Faithfulness = calculateDimensionMetrics(faithScores)
	metrics.AnswerRelevance = calculateDimensionMetrics(ansRelScores)
	metrics.Correctness = calculateDimensionMetrics(correctScores)
	metrics.ContextRelevance = calculateDimensionMetrics(ctxRelScores)

	return metrics
}

// calculateDimensionMetrics computes statistics for a single dimension
func calculateDimensionMetrics(scores []int) DimensionMetrics {
	metrics := DimensionMetrics{
		ScoreDistribution:   make(map[int]int),
		AccuracyAtThreshold: make(map[int]float64),
	}

	if len(scores) == 0 {
		return metrics
	}

	// Calculate total and distribution
	totalScore := 0
	for _, score := range scores {
		totalScore += score
		metrics.ScoreDistribution[score]++
	}

	// Calculate average
	metrics.AverageScore = float64(totalScore) / float64(len(scores))

	// Calculate median
	sortedScores := make([]int, len(scores))
	copy(sortedScores, scores)
	sort.Ints(sortedScores)
	if len(sortedScores)%2 == 0 {
		mid := len(sortedScores) / 2
		metrics.MedianScore = float64(sortedScores[mid-1]+sortedScores[mid]) / 2.0
	} else {
		metrics.MedianScore = float64(sortedScores[len(sortedScores)/2])
	}

	// Calculate accuracy at each threshold
	for threshold := 1; threshold <= 5; threshold++ {
		count := 0
		for _, score := range scores {
			if score >= threshold {
				count++
			}
		}
		metrics.AccuracyAtThreshold[threshold] = float64(count) / float64(len(scores))
	}

	// Calculate pass rate at threshold 4
	metrics.PassRateAtFour = metrics.AccuracyAtThreshold[4]

	return metrics
}
