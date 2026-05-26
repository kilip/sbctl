package memory

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kilip/sbctl/ent"
)

func TestExtractFacts(t *testing.T) {
	ctx := context.Background()

	t.Run("trivial message produces zero memory rows", func(t *testing.T) {
		msgContent, _ := json.Marshal(map[string]string{"text": "ok thanks"})
		msg := &ent.Message{
			Content: msgContent,
		}

		facts, err := ExtractFacts(ctx, nil, msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(facts) != 0 {
			t.Errorf("expected 0 facts for trivial message, got %d", len(facts))
		}
	})

	t.Run("valid fact message produces extracted fact", func(t *testing.T) {
		msgContent, _ := json.Marshal(map[string]string{"text": "My name is Toni and I live in Jakarta"})
		msg := &ent.Message{
			Content: msgContent,
		}

		facts, err := ExtractFacts(ctx, nil, msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(facts) != 1 {
			t.Fatalf("expected 1 fact, got %d", len(facts))
		}

		expectedContent := "My name is Toni and I live in Jakarta"
		if facts[0].Content != expectedContent {
			t.Errorf("expected fact content %q, got %q", expectedContent, facts[0].Content)
		}
	})
}
