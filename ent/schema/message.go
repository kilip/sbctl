package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
	"github.com/kilip/sbctl/ent/mixin"
)

// Message holds the schema definition for the Message entity.
type Message struct {
	ent.Schema
}

// Fields of the Message.
func (Message) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(func() uuid.UUID {
				id, _ := uuid.NewV7()
				return id
			}),
		field.UUID("session_id", uuid.UUID{}),
		field.Enum("role").
			Values("system", "user", "assistant", "tool"),
		field.JSON("content", json.RawMessage{}),
		field.Enum("status").
			Values("pending", "awaiting_approval", "done").
			Default("pending"),
		field.Int("token_count").
			Optional(),
		field.JSON("metadata", json.RawMessage{}).
			Optional(),
		field.Time("created").
			Default(time.Now),
		field.Time("updated").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Mixin of the Message.
func (Message) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.SoftDeleteMixin{},
	}
}

// Edges of the Message.
func (Message) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("session", Session.Type).
			Ref("messages").
			Field("session_id").
			Unique().
			Required(),
		edge.To("memories", Memory.Type),
	}
}
