package rag

import (
	"strings"
)

// ChunkText splits the input text into chunks based on paragraph boundaries
func ChunkText(text string) []string {
	if strings.TrimSpace(text) == "" {
		return []string{}
	}

	// Split by paragraph boundaries (assumes double newlines)
	paragraphs := strings.Split(text, "\n\n")

	// Prepare chunks: trim whitespace and filter empty paragraphs
	var chunks []string
	for _, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			chunks = append(chunks, trimmed)
		}
	}

	return chunks
}
