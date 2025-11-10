package rag

import (
	"reflect"
	"strings"
	"testing"
)

func TestChunking(t *testing.T) {
	type testCase struct {
		input    string
		expected []string
	}

	testCases := []testCase{
		{
			input:    "",
			expected: []string{},
		},
		{
			input: "a single line",
			expected: []string{
				"a single line",
			},
		},
		{
			input: "a single line\nanother line in same paragraph",
			expected: []string{
				"a single line\nanother line in same paragraph",
			},
		},
		{
			input: "a single line in first paragraph\n\nanother line in another paragraph",
			expected: []string{
				"a single line in first paragraph\n\nanother line in another paragraph",
			},
		},
	}

	for _, test := range testCases {
		actual := ChunkText(test.input)
		if len(actual) != len(test.expected) {
			t.Errorf("Expected %v chunks, but got %v", len(test.expected), len(actual))
			continue
		}
		if !reflect.DeepEqual(actual, test.expected) {
			t.Errorf("Expected %v to equal %v", actual, test.expected)
		}
	}
}

func TestChunkingAccumulation(t *testing.T) {
	// Test that multiple small paragraphs get combined into one chunk
	t.Run("multiple small paragraphs combine", func(t *testing.T) {
		input := "First paragraph with some text.\n\nSecond paragraph with more text.\n\nThird paragraph also here."
		chunks := ChunkText(input)

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk for small paragraphs, got %d", len(chunks))
		}

		expected := "First paragraph with some text.\n\nSecond paragraph with more text.\n\nThird paragraph also here."
		if chunks[0] != expected {
			t.Errorf("Expected combined chunk, got: %v", chunks[0])
		}
	})

	// Test that a single long paragraph exceeding target becomes one chunk
	t.Run("single long paragraph stays together", func(t *testing.T) {
		longPara := strings.Repeat("This is a very long paragraph that exceeds the target chunk size. ", 20)
		chunks := ChunkText(longPara)

		if len(chunks) != 1 {
			t.Errorf("Expected 1 chunk for long paragraph, got %d", len(chunks))
		}

		if len(chunks[0]) <= TargetChunkSize {
			t.Errorf("Expected chunk length > %d, got %d", TargetChunkSize, len(chunks[0]))
		}
	})

	// Test that paragraphs split when combined length would exceed target
	t.Run("paragraphs split appropriately", func(t *testing.T) {
		// Create paragraphs that will require splitting
		para1 := strings.Repeat("First chunk content. ", 30)  // ~600 chars
		para2 := strings.Repeat("Second chunk content. ", 30) // ~660 chars
		para3 := strings.Repeat("Third chunk content. ", 30)  // ~630 chars

		input := para1 + "\n\n" + para2 + "\n\n" + para3
		chunks := ChunkText(input)

		// Should create multiple chunks since each pair would exceed 1000 chars
		if len(chunks) < 2 {
			t.Errorf("Expected at least 2 chunks, got %d", len(chunks))
		}

		// Verify all chunks are reasonable size
		for i, chunk := range chunks {
			// Each chunk should either be <= target or be a single long paragraph
			if len(chunk) > TargetChunkSize {
				// Check it's a single paragraph (no \n\n separator)
				if strings.Contains(chunk, "\n\n") {
					t.Errorf("Chunk %d exceeds target but contains multiple paragraphs", i)
				}
			}
		}
	})
}
