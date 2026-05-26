package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Memory holds the schema definition for the Memory entity.
type Memory struct {
	ent.Schema
}

// Fields of the Memory.
func (Memory) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(func() uuid.UUID {
				id, _ := uuid.NewV7()
				return id
			}),
		field.UUID("user_id", uuid.UUID{}),
		field.UUID("session_id", uuid.UUID{}),
		field.UUID("message_id", uuid.UUID{}),
		field.Text("content"),
		field.JSON("metadata", json.RawMessage{}).
			Optional(),
		field.Bytes("embedding").
			Optional().
			Nillable(),
		field.Time("created").
			Default(time.Now),
		field.Time("updated").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Memory.
func (Memory) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("memories").
			Field("user_id").
			Unique().
			Required(),
		edge.From("session", Session.Type).
			Ref("memories").
			Field("session_id").
			Unique().
			Required(),
		edge.From("message", Message.Type).
			Ref("memories").
			Field("message_id").
			Unique().
			Required(),
	}
}

// Indexes of the Memory.
func (Memory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
	}
}
