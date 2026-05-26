package storage

import (
	"context"
	"encoding/json"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/sbctl/ent"
	"github.com/kilip/sbctl/ent/account"
	"github.com/kilip/sbctl/ent/profile"
	"github.com/kilip/sbctl/ent/session"
)

// CreateSession creates a new session.
func CreateSession(ctx context.Context, client *ent.Client, accountID uuid.UUID, profileID *uuid.UUID, title, summary string, metadata json.RawMessage) (*ent.Session, error) {
	builder := client.Session.Create().
		SetAccountID(accountID)
	if profileID != nil {
		builder.SetProfileID(*profileID)
	}
	if title != "" {
		builder.SetTitle(title)
	}
	if summary != "" {
		builder.SetSummary(summary)
	}
	if len(metadata) > 0 {
		builder.SetMetadata(metadata)
	}
	return builder.Save(ctx)
}

// GetSessionByID retrieves a session by ID.
func GetSessionByID(ctx context.Context, client *ent.Client, id uuid.UUID) (*ent.Session, error) {
	return client.Session.Query().Where(session.ID(id)).Only(ctx)
}

// GetSessionByKey retrieves a session by a deterministic session key in metadata.
func GetSessionByKey(ctx context.Context, client *ent.Client, key string) (*ent.Session, error) {
	return client.Session.Query().
		Where(func(s *sql.Selector) {
			s.Where(sql.ExprP("json_extract(metadata, '$.session_key') = ?", key))
		}).
		Only(ctx)
}

// SoftDeleteSession marks a session as deleted.
func SoftDeleteSession(ctx context.Context, client *ent.Client, id uuid.UUID) error {
	_, err := client.Session.UpdateOneID(id).
		SetDeleted(time.Now()).
		Save(ctx)
	return err
}

// ResolveSessionProfileDir resolves the working directory path for a session.
// If profile_id is NULL, it falls back to the user's default profile.
func ResolveSessionProfileDir(ctx context.Context, client *ent.Client, sess *ent.Session) (string, error) {
	if sess.ProfileID != nil && *sess.ProfileID != uuid.Nil {
		p, err := client.Profile.Query().Where(profile.ID(*sess.ProfileID)).Only(ctx)
		if err != nil {
			return "", err
		}
		return p.WorkingDir, nil
	}

	// Null profile_id: fallback to user default profile
	acc, err := client.Account.Query().Where(account.ID(sess.AccountID)).Only(ctx)
	if err != nil {
		return "", err
	}
	p, err := GetDefaultProfile(ctx, client, acc.UserID)
	if err != nil {
		return "", err
	}
	return p.WorkingDir, nil
}
