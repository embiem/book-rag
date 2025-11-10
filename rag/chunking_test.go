package rag

import (
	"reflect"
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
				"a single line in first paragraph",
				"another line in another paragraph",
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
