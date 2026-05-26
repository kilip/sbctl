package storage

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/kilip/sbctl/ent"
)

// CreateAuditLog inserts a new audit log record. AuditLog is append-only.
func CreateAuditLog(ctx context.Context, client *ent.Client, userID uuid.UUID, action, entityType string, entityID uuid.UUID, metadata json.RawMessage) (*ent.AuditLog, error) {
	builder := client.AuditLog.Create().
		SetUserID(userID).
		SetAction(action).
		SetEntityType(entityType).
		SetEntityID(entityID)
	if len(metadata) > 0 {
		builder.SetMetadata(metadata)
	}
	return builder.Save(ctx)
}
