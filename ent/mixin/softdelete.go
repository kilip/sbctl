package mixin

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

type SoftDeleteCtxKey struct{}

// SkipSoftDelete returns a context that skips the soft delete filter.
func SkipSoftDelete(ctx context.Context) context.Context {
	return context.WithValue(ctx, SoftDeleteCtxKey{}, true)
}

// SoftDeleteMixin implements soft deletion.
type SoftDeleteMixin struct {
	mixin.Schema
}

// Fields of the SoftDeleteMixin.
func (SoftDeleteMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time("deleted").
			Optional().
			Nillable(),
	}
}
