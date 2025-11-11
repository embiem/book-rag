package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/embiem/book-rag/db"
	"github.com/embiem/book-rag/eval"
	"github.com/openai/openai-go/v3"
)

func main() {
	// Parse command-line flags
	bookID := flag.Int64("book-id", 0, "Book ID to generate QA pairs from (0 = use all books)")
	samples := flag.Int("samples", 250, "Number of QA pairs to generate (before filtering)")
	output := flag.String("output", "testdata/eval_dataset.json", "Output file path")
	flag.Parse()

	if *samples <= 0 {
		log.Fatal("samples must be positive")
	}

	// Initialize database
	if err := db.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize OpenAI client
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}
	client := openai.NewClient()

	ctx := context.Background()

	// Fetch book passages (chunks)
	fmt.Println("Fetching book passages from database...")

	var chunks []eval.ChunkWithBook
	var err error

	if *bookID > 0 {
		passages, err := db.Queries.GetBookPassages(ctx, *bookID)
		if err != nil {
			log.Fatalf("Failed to fetch passages for book %d: %v", *bookID, err)
		}
		fmt.Printf("Found %d passages from book %d\n", len(passages), *bookID)
		chunks = make([]eval.ChunkWithBook, len(passages))
		for i, p := range passages {
			chunks[i] = eval.ChunkWithBook{
				Text:   p.PassageText,
				BookID: p.BookID,
			}
		}
	} else {
		passages, err := db.Queries.GetAllBookPassages(ctx)
		if err != nil {
			log.Fatalf("Failed to fetch passages: %v", err)
		}
		fmt.Printf("Found %d passages from all books\n", len(passages))
		chunks = make([]eval.ChunkWithBook, len(passages))
		for i, p := range passages {
			chunks[i] = eval.ChunkWithBook{
				Text:   p.PassageText,
				BookID: p.BookID,
			}
		}
	}

	if len(chunks) == 0 {
		log.Fatal("No passages found. Please ingest at least one book first.")
	}

	// Generate dataset
	fmt.Printf("\nGenerating evaluation dataset with target of %d samples...\n\n", *samples)
	dataset, err := eval.GenerateDataset(ctx, &client, chunks, *samples)
	if err != nil {
		log.Fatalf("Failed to generate dataset: %v", err)
	}

	if len(dataset.QAPairs) == 0 {
		log.Fatal("No QA pairs passed quality filtering. Try generating more samples.")
	}

	// Save to file
	fmt.Printf("\nSaving dataset to %s...\n", *output)

	// Create directory if it doesn't exist
	if err := os.MkdirAll("testdata", 0755); err != nil {
		log.Fatalf("Failed to create testdata directory: %v", err)
	}

	file, err := os.Create(*output)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(dataset); err != nil {
		log.Fatalf("Failed to write dataset: %v", err)
	}

	fmt.Printf("\n✓ Dataset generation complete!\n")
	fmt.Printf("  Generated: %d QA pairs\n", len(dataset.QAPairs))
	fmt.Printf("  Output: %s\n", *output)
}
