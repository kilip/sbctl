package storage

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/sbctl/ent"
	"github.com/kilip/sbctl/ent/account"
	"github.com/kilip/sbctl/ent/user"
)

// CreateUser creates a new user.
func CreateUser(ctx context.Context, client *ent.Client, name string, role string) (*ent.User, error) {
	return client.User.Create().
		SetName(name).
		SetRole(user.Role(role)).
		Save(ctx)
}

// GetUserByID retrieves a user by ID.
func GetUserByID(ctx context.Context, client *ent.Client, id uuid.UUID) (*ent.User, error) {
	return client.User.Query().Where(user.ID(id)).Only(ctx)
}

// SoftDeleteUser marks a user as deleted.
func SoftDeleteUser(ctx context.Context, client *ent.Client, id uuid.UUID) error {
	_, err := client.User.UpdateOneID(id).
		SetDeleted(time.Now()).
		Save(ctx)
	return err
}

// CreateAccount creates a new platform account linked to a user.
func CreateAccount(ctx context.Context, client *ent.Client, userID uuid.UUID, platform, externalID, status string, metadata json.RawMessage) (*ent.Account, error) {
	return client.Account.Create().
		SetUserID(userID).
		SetPlatform(platform).
		SetExternalID(externalID).
		SetStatus(account.Status(status)).
		SetMetadata(metadata).
		Save(ctx)
}

// GetAccountByPlatformID retrieves an account by platform and external ID.
func GetAccountByPlatformID(ctx context.Context, client *ent.Client, platform, externalID string) (*ent.Account, error) {
	return client.Account.Query().
		Where(
			account.Platform(platform),
			account.ExternalID(externalID),
		).
		Only(ctx)
}

// SoftDeleteAccount marks an account as deleted.
func SoftDeleteAccount(ctx context.Context, client *ent.Client, id uuid.UUID) error {
	_, err := client.Account.UpdateOneID(id).
		SetDeleted(time.Now()).
		Save(ctx)
	return err
}
