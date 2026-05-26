package memory

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/kilip/sbctl/ent"
)

type Fact struct {
	Content  string
	Metadata map[string]any
}

// ExtractFacts evaluates whether the message contains facts worth persisting.
func ExtractFacts(ctx context.Context, session *ent.Session, msg *ent.Message) ([]Fact, error) {
	// Parse content text
	var contentMap map[string]any
	if err := json.Unmarshal(msg.Content, &contentMap); err != nil {
		return nil, nil // not a text message or invalid content structure
	}

	text, ok := contentMap["text"].(string)
	if !ok || text == "" {
		return nil, nil
	}

	textLower := strings.ToLower(strings.TrimSpace(text))

	// Filter out trivial sentences
	if textLower == "ok thanks" || textLower == "thanks" || textLower == "hello" || textLower == "hi" || textLower == "ok" {
		return nil, nil
	}

	// Heuristic fact detection for MVP rule-based extraction
	var facts []Fact
	if strings.Contains(textLower, "my name is") ||
		strings.Contains(textLower, "i live in") ||
		strings.Contains(textLower, "remember that") ||
		strings.Contains(textLower, "status of") ||
		strings.Contains(textLower, "project") {

		facts = append(facts, Fact{
			Content: text,
			Metadata: map[string]any{
				"extracted_from": "text",
				"type":           "semantic_fact",
			},
		})
	}

	return facts, nil
}
