package compact

import "strings"

// Estimate counts tokens as about four bytes each.
func Estimate(text string) int {
	if text == "" {
		return 0
	}
	return (len(text) + 3) / 4
}

// Summarize splits text into token-sized chunks and keeps a short head from each.
// The result is taken from the source text. An empty source returns an empty summary.
func Summarize(text string, chunkTokens int) (summary string, chunks int) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", 0
	}
	if chunkTokens < 8 {
		chunkTokens = 8
	}
	size := chunkTokens * 4
	var parts []string
	for i := 0; i < len(text); i += size {
		end := i + size
		if end > len(text) {
			end = len(text)
		}
		chunk := strings.TrimSpace(text[i:end])
		if chunk == "" {
			continue
		}
		head := chunk
		if len(head) > 80 {
			head = head[:80]
		}
		parts = append(parts, head)
	}
	if len(parts) == 0 {
		return "", 0
	}
	return strings.Join(parts, "\n"), len(parts)
}
