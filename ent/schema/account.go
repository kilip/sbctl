package schema

import (
	"encoding/json"
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/kilip/sbctl/ent/mixin"
)

// Account holds the schema definition for the Account entity.
type Account struct {
	ent.Schema
}

// Fields of the Account.
func (Account) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(func() uuid.UUID {
				id, _ := uuid.NewV7()
				return id
			}),
		field.UUID("user_id", uuid.UUID{}),
		field.String("platform"),
		field.String("external_id"),
		field.Enum("status").
			Values("active", "banned", "deactivated").
			Default("active"),
		field.JSON("metadata", json.RawMessage{}).
			Optional(),
		field.Time("created").
			Default(time.Now),
		field.Time("updated").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Mixin of the Account.
func (Account) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.SoftDeleteMixin{},
	}
}

// Edges of the Account.
func (Account) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("accounts").
			Field("user_id").
			Unique().
			Required(),
		edge.To("sessions", Session.Type),
	}
}

// Indexes of the Account.
func (Account) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("external_id"),
	}
}
