package storage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/sbctl/ent"
	"github.com/kilip/sbctl/ent/profile"
)

// CreateProfile creates a new profile.
func CreateProfile(ctx context.Context, client *ent.Client, userID uuid.UUID, name string, workingDir string, isDefault bool) (*ent.Profile, error) {
	return client.Profile.Create().
		SetUserID(userID).
		SetName(name).
		SetWorkingDir(workingDir).
		SetIsDefault(isDefault).
		Save(ctx)
}

// GetDefaultProfile retrieves the default profile for a user.
func GetDefaultProfile(ctx context.Context, client *ent.Client, userID uuid.UUID) (*ent.Profile, error) {
	return client.Profile.Query().
		Where(
			profile.UserID(userID),
			profile.IsDefault(true),
		).
		Only(ctx)
}

// SetDefaultProfile atomically marks a profile as default and sets others to false.
func SetDefaultProfile(ctx context.Context, client *ent.Client, profileID uuid.UUID) error {
	_, err := client.Profile.UpdateOneID(profileID).
		SetIsDefault(true).
		Save(ctx)
	return err
}

// SoftDeleteProfile marks a profile as deleted. Default profile cannot be deleted.
func SoftDeleteProfile(ctx context.Context, client *ent.Client, profileID uuid.UUID) error {
	p, err := client.Profile.Query().Where(profile.ID(profileID)).Only(ctx)
	if err != nil {
		return err
	}
	if p.IsDefault {
		return errors.New("cannot delete the default profile")
	}

	_, err = client.Profile.UpdateOneID(profileID).
		SetDeleted(time.Now()).
		Save(ctx)
	return err
}
