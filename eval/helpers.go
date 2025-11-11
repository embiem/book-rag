package eval

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// parseScoreFromResponse extracts a 1-5 score and reasoning from an LLM response.
// It expects the response to contain "Score: X" where X is a digit 1-5.
// The reasoning is extracted from everything before "Score:".
func parseScoreFromResponse(response string) (score int, reasoning string, err error) {
	// Extract score using regex
	scoreRegex := regexp.MustCompile(`(?i)score:\s*(\d)`)
	matches := scoreRegex.FindStringSubmatch(response)
	if len(matches) < 2 {
		return 0, "", fmt.Errorf("could not parse score from response: %s", response)
	}

	score, err = strconv.Atoi(matches[1])
	if err != nil || score < 1 || score > 5 {
		return 0, "", fmt.Errorf("invalid score value: %s", matches[1])
	}

	// Extract reasoning (everything before "Score:")
	reasoning = strings.Split(response, "Score:")[0]
	reasoning = strings.TrimSpace(reasoning)

	return score, reasoning, nil
}
