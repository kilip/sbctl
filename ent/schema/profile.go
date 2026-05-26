package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
	"github.com/kilip/sbctl/ent/mixin"
)

// Profile holds the schema definition for the Profile entity.
type Profile struct {
	ent.Schema
}

// Fields of the Profile.
func (Profile) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(func() uuid.UUID {
				id, _ := uuid.NewV7()
				return id
			}),
		field.UUID("user_id", uuid.UUID{}),
		field.String("name"),
		field.String("working_dir"),
		field.Bool("is_default").
			Default(false),
		field.Time("created").
			Default(time.Now),
		field.Time("updated").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Mixin of the Profile.
func (Profile) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixin.SoftDeleteMixin{},
	}
}

// Edges of the Profile.
func (Profile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("profiles").
			Field("user_id").
			Unique().
			Required(),
		edge.To("sessions", Session.Type),
	}
}

// Indexes of the Profile.
func (Profile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "name").Unique(),
	}
}
