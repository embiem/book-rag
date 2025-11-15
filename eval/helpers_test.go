package eval

import (
	"testing"
)

func TestParseScoreFromResponse(t *testing.T) {
	tests := []struct {
		name          string
		response      string
		expectedScore int
		shouldError   bool
	}{
		{
			name:          "Standard format",
			response:      "This is great reasoning.\n\nScore: 5",
			expectedScore: 5,
			shouldError:   false,
		},
		{
			name:          "Markdown bold format",
			response:      "This is great reasoning.\n\n**Score**: 4",
			expectedScore: 4,
			shouldError:   false,
		},
		{
			name:          "Extra spaces",
			response:      "This is great reasoning.\n\nScore : 3",
			expectedScore: 3,
			shouldError:   false,
		},
		{
			name:          "No space after colon",
			response:      "This is great reasoning.\n\nScore:2",
			expectedScore: 2,
			shouldError:   false,
		},
		{
			name:          "Case insensitive",
			response:      "This is great reasoning.\n\nscore: 1",
			expectedScore: 1,
			shouldError:   false,
		},
		{
			name:        "Missing score",
			response:    "This is great reasoning but no score",
			shouldError: true,
		},
		{
			name:        "Invalid score (too high)",
			response:    "This is great reasoning.\n\nScore: 6",
			shouldError: true,
		},
		{
			name:        "Invalid score (zero)",
			response:    "This is great reasoning.\n\nScore: 0",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, reasoning, err := parseScoreFromResponse(tt.response)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if score != tt.expectedScore {
				t.Errorf("Expected score %d, got %d", tt.expectedScore, score)
			}

			if reasoning == "" {
				t.Errorf("Expected reasoning to be extracted, got empty string")
			}
		})
	}
}
