package storage

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/kilip/sbctl/ent"
	"github.com/kilip/sbctl/ent/message"
)

// CreateMessage creates a new message.
func CreateMessage(ctx context.Context, client *ent.Client, sessionID uuid.UUID, role string, content json.RawMessage, status string, tokenCount int, metadata json.RawMessage) (*ent.Message, error) {
	builder := client.Message.Create().
		SetSessionID(sessionID).
		SetRole(message.Role(role)).
		SetContent(content).
		SetStatus(message.Status(status))
	if tokenCount > 0 {
		builder.SetTokenCount(tokenCount)
	}
	if len(metadata) > 0 {
		builder.SetMetadata(metadata)
	}
	return builder.Save(ctx)
}

// GetMessagesBySession retrieves messages in chronological order.
// If limit > 0, returns the N most recent messages.
func GetMessagesBySession(ctx context.Context, client *ent.Client, sessionID uuid.UUID, limit int) ([]*ent.Message, error) {
	query := client.Message.Query().
		Where(message.SessionID(sessionID))

	if limit > 0 {
		// Retrieve the N most recent messages ordered by ID descending
		msgs, err := query.Order(ent.Desc(message.FieldID)).
			Limit(limit).
			All(ctx)
		if err != nil {
			return nil, err
		}
		// Reverse the order to chronological ascending
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}
		return msgs, nil
	}

	// Returns all messages in chronological ascending order
	return query.Order(ent.Asc(message.FieldID)).
		All(ctx)
}

// UpdateMessageStatus updates the status of a message.
func UpdateMessageStatus(ctx context.Context, client *ent.Client, id uuid.UUID, status string) (*ent.Message, error) {
	return client.Message.UpdateOneID(id).
		SetStatus(message.Status(status)).
		Save(ctx)
}
