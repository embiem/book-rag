package eval

import (
	"sort"
)

// CalculateMetrics computes aggregate statistics from evaluation results
func CalculateMetrics(results []EvalResult) EvalMetrics {
	metrics := EvalMetrics{
		TotalQuestions:      len(results),
		ScoreDistribution:   make(map[int]int),
		AccuracyAtThreshold: make(map[int]float64),
	}

	if len(results) == 0 {
		return metrics
	}

	// Collect scores, excluding failed results
	scores := make([]int, 0, len(results))
	totalScore := 0
	failedCount := 0
	for _, result := range results {
		if result.Failed {
			failedCount++
			continue // Exclude failed results from scoring metrics
		}
		score := result.CorrectnessScore
		scores = append(scores, score)
		totalScore += score
		metrics.ScoreDistribution[score]++
	}

	metrics.FailedQuestions = failedCount

	// If all results failed, return empty metrics
	if len(scores) == 0 {
		return metrics
	}

	// Calculate average (only from successful results)
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

	// Calculate accuracy at each threshold (percentage >= threshold, excluding failed results)
	for threshold := 1; threshold <= 5; threshold++ {
		count := 0
		for _, score := range scores {
			if score >= threshold {
				count++
			}
		}
		metrics.AccuracyAtThreshold[threshold] = float64(count) / float64(len(scores))
	}

	// Calculate pass rate at threshold 4 (percentage scoring >= 4)
	metrics.PassRateAtFour = metrics.AccuracyAtThreshold[4]

	return metrics
}
