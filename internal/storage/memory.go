package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/sbctl/ent"
	"github.com/kilip/sbctl/ent/memory"
)

// SaveMemory persists a new fact extracted from a message.
func SaveMemory(ctx context.Context, client *ent.Client, userID, sessionID, messageID uuid.UUID, content string, metadata json.RawMessage) (*ent.Memory, error) {
	builder := client.Memory.Create().
		SetUserID(userID).
		SetSessionID(sessionID).
		SetMessageID(messageID).
		SetContent(content)
	if len(metadata) > 0 {
		builder.SetMetadata(metadata)
	}
	return builder.Save(ctx)
}

// GetMemoryByUser retrieves memories for a user, optionally filtered by key-value metadata pairs.
func GetMemoryByUser(ctx context.Context, client *ent.Client, userID uuid.UUID, filter map[string]string) ([]*ent.Memory, error) {
	query := client.Memory.Query().
		Where(memory.UserID(userID))

	for k, v := range filter {
		keyPath := fmt.Sprintf("$.%s", k)
		val := v
		query = query.Where(func(s *sql.Selector) {
			s.Where(sql.ExprP("json_extract(metadata, ?) = ?", keyPath, val))
		})
	}

	return query.All(ctx)
}

// DeleteMemory physically deletes a memory record from the database.
func DeleteMemory(ctx context.Context, client *ent.Client, id uuid.UUID) error {
	return client.Memory.DeleteOneID(id).Exec(ctx)
}
