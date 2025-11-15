package eval

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// parseScoreFromResponse extracts a 1-5 score and reasoning from an LLM response.
// It handles various formatting variations including markdown bold (**Score**:).
// The reasoning is extracted from everything before the score.
func parseScoreFromResponse(response string) (score int, reasoning string, err error) {
	// Extract score using regex - handles variations like:
	// "Score: 3", "**Score**: 3", "Score : 3", "Score:3"
	scoreRegex := regexp.MustCompile(`(?i)\*{0,2}score\*{0,2}\s*:\s*(\d)`)
	matches := scoreRegex.FindStringSubmatch(response)
	if len(matches) < 2 {
		return 0, "", fmt.Errorf("could not parse score from response: %s", response)
	}

	score, err = strconv.Atoi(matches[1])
	if err != nil || score < 1 || score > 5 {
		return 0, "", fmt.Errorf("invalid score value: %s", matches[1])
	}

	// Extract reasoning (everything before the score pattern)
	// Split on the pattern we matched
	reasoningSplitRegex := regexp.MustCompile(`(?i)\*{0,2}score\*{0,2}\s*:`)
	parts := reasoningSplitRegex.Split(response, 2)
	if len(parts) > 0 {
		reasoning = strings.TrimSpace(parts[0])
	}

	return score, reasoning, nil
}
