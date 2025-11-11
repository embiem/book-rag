package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/embiem/book-rag/eval"
	"github.com/openai/openai-go/v3"
)

func main() {
	// Parse command-line flags
	datasetFile := flag.String("dataset", "testdata/eval_dataset.json", "Path to evaluation dataset")
	outputFile := flag.String("output", "testdata/results/baseline.json", "Path for results output")
	ragURL := flag.String("rag-url", "http://localhost:3000", "Base URL of RAG server")
	flag.Parse()

	// Load dataset
	fmt.Printf("Loading dataset from %s...\n", *datasetFile)
	file, err := os.Open(*datasetFile)
	if err != nil {
		log.Fatalf("Failed to open dataset file: %v", err)
	}
	defer file.Close()

	var dataset eval.EvalDataset
	if err := json.NewDecoder(file).Decode(&dataset); err != nil {
		log.Fatalf("Failed to parse dataset: %v", err)
	}

	fmt.Printf("Loaded %d QA pairs\n", len(dataset.QAPairs))

	if len(dataset.QAPairs) == 0 {
		log.Fatal("Dataset is empty")
	}

	// Initialize OpenAI client for judging
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}
	client := openai.NewClient()

	ctx := context.Background()

	// Run evaluation
	fmt.Printf("\nStarting evaluation against RAG server at %s...\n\n", *ragURL)
	run, err := eval.RunEvaluation(ctx, &client, &dataset, *ragURL)
	if err != nil {
		log.Fatalf("Evaluation failed: %v", err)
	}

	run.DatasetFile = *datasetFile

	// Display metrics
	separator := strings.Repeat("=", 60)
	fmt.Printf("\n%s\n", separator)
	fmt.Printf("EVALUATION RESULTS\n")
	fmt.Printf("%s\n\n", separator)
	fmt.Printf("Total Questions:     %d\n", run.Metrics.TotalQuestions)
	if run.Metrics.FailedQuestions > 0 {
		fmt.Printf("Failed Questions:    %d\n", run.Metrics.FailedQuestions)
		fmt.Printf("Successful:          %d\n", run.Metrics.TotalQuestions-run.Metrics.FailedQuestions)
	}
	fmt.Printf("Average Score:       %.2f / 5.0\n", run.Metrics.AverageScore)
	fmt.Printf("Median Score:        %.1f\n", run.Metrics.MedianScore)
	fmt.Printf("Pass Rate (≥4):      %.3f\n", run.Metrics.PassRateAtFour)
	fmt.Printf("\nScore Distribution:\n")
	for score := 5; score >= 1; score-- {
		count := run.Metrics.ScoreDistribution[score]
		pct := float64(count) / float64(run.Metrics.TotalQuestions) * 100
		fmt.Printf("  %d: %3d (%.1f%%)\n", score, count, pct)
	}
	fmt.Printf("\nAccuracy by Threshold:\n")
	for threshold := 5; threshold >= 1; threshold-- {
		pct := run.Metrics.AccuracyAtThreshold[threshold] * 100
		fmt.Printf("  ≥%d: %.1f%%\n", threshold, pct)
	}
	fmt.Printf("\n%s\n\n", separator)

	// Save results
	fmt.Printf("Saving results to %s...\n", *outputFile)

	// Create directory if it doesn't exist
	if err := os.MkdirAll("testdata/results", 0755); err != nil {
		log.Fatalf("Failed to create results directory: %v", err)
	}

	outFile, err := os.Create(*outputFile)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	encoder := json.NewEncoder(outFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(run); err != nil {
		log.Fatalf("Failed to write results: %v", err)
	}

	fmt.Printf("\n✓ Evaluation complete! Results saved to %s\n", *outputFile)
}
