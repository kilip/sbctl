package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// AuditLog holds the schema definition for the AuditLog entity.
type AuditLog struct {
	ent.Schema
}

// Fields of the AuditLog.
func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(func() uuid.UUID {
				id, _ := uuid.NewV7()
				return id
			}),
		field.UUID("user_id", uuid.UUID{}),
		field.String("action"),
		field.String("entity_type"),
		field.UUID("entity_id", uuid.UUID{}),
		field.JSON("metadata", json.RawMessage{}).
			Optional(),
		field.Time("created").
			Default(time.Now),
	}
}

// Edges of the AuditLog.
func (AuditLog) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("audit_logs").
			Field("user_id").
			Unique().
			Required(),
	}
}
