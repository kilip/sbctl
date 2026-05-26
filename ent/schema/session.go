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

// Session holds the schema definition for the Session entity.
type Session struct {
	ent.Schema
}

// Fields of the Session.
func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(func() uuid.UUID {
				id, _ := uuid.NewV7()
				return id
			}),
		field.UUID("account_id", uuid.UUID{}),
		field.UUID("profile_id", uuid.UUID{}).
			Optional().
			Nillable(),
		field.String("title").
			Optional().
			Nillable(),
		field.Text("summary").
			Optional().
			Nillable(),
		field.JSON("metadata", json.RawMessage{}).
			Optional(),
		field.Time("created").
			Default(time.Now),
		field.Time("updated").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Mixin of the Session.
func (Session) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.SoftDeleteMixin{},
	}
}

// Edges of the Session.
func (Session) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", Account.Type).
			Ref("sessions").
			Field("account_id").
			Unique().
			Required(),
		edge.From("profile", Profile.Type).
			Ref("sessions").
			Field("profile_id").
			Unique(),
		edge.To("messages", Message.Type),
		edge.To("memories", Memory.Type),
	}
}
