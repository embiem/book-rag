package rag

import (
	"strings"
)

// Chunks get filled with paragraphs up to this amount of chars
const TargetChunkSize = 1000

func ChunkText(text string) []string {
	if strings.TrimSpace(text) == "" {
		return []string{}
	}

	// Split by paragraph boundaries (assumes double newlines)
	paragraphs := strings.Split(text, "\n\n")

	// Filter and trim paragraphs
	var cleanParagraphs []string
	for _, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			cleanParagraphs = append(cleanParagraphs, trimmed)
		}
	}

	// Accumulate paragraphs into chunks up to TargetChunkSize
	var chunks []string
	var currentChunk strings.Builder

	for _, para := range cleanParagraphs {
		// If this is the first paragraph in the chunk, add it
		if currentChunk.Len() == 0 {
			currentChunk.WriteString(para)
		} else {
			// Check if adding this paragraph would exceed the target size
			potentialLength := currentChunk.Len() + 2 + len(para) // +2 for "\n\n"
			if potentialLength <= TargetChunkSize {
				currentChunk.WriteString("\n\n")
				currentChunk.WriteString(para)
			} else {
				// Finalize current chunk and start a new one
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentChunk.WriteString(para)
			}
		}
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}
