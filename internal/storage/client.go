package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	entSql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/sbctl/ent"
	"github.com/kilip/sbctl/ent/account"
	"github.com/kilip/sbctl/ent/message"
	"github.com/kilip/sbctl/ent/mixin"
	"github.com/kilip/sbctl/ent/profile"
	_ "github.com/kilip/sbctl/ent/runtime"
	"github.com/kilip/sbctl/ent/session"
	"github.com/kilip/sbctl/ent/user"
	"github.com/kilip/sbctl/internal/config"
	_ "modernc.org/sqlite"
)

// Open initializes and returns an ent.Client configured for pure Go SQLite with WAL mode.
func Open(cfg *config.Config) (*ent.Client, error) {
	dbPath := cfg.Db.Path
	if strings.HasPrefix(dbPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home dir: %w", err)
		}
		dbPath = filepath.Join(home, dbPath[2:])
	}

	var dsn string
	if strings.HasPrefix(dbPath, "file:") {
		dsn = dbPath
		if !strings.Contains(dsn, "foreign_keys") {
			if strings.Contains(dsn, "?") {
				dsn += "&_pragma=foreign_keys(1)"
			} else {
				dsn += "?_pragma=foreign_keys(1)"
			}
		}
		if !strings.Contains(dsn, "journal_mode") {
			if strings.Contains(dsn, "?") {
				dsn += "&_journal_mode=WAL"
			} else {
				dsn += "?_journal_mode=WAL"
			}
		}
	} else if dbPath == ":memory:" {
		dsn = "file::memory:?cache=shared&_journal_mode=WAL&_pragma=foreign_keys(1)"
	} else {
		// Ensure directory exists
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
		dsn = fmt.Sprintf("file:%s?cache=shared&_journal_mode=WAL&_pragma=foreign_keys(1)", dbPath)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Wrap standard sql.DB into ent sql.Driver
	drv := entSql.OpenDB("sqlite3", db)
	client := ent.NewClient(ent.Driver(drv))

	// Register all runtime hooks
	registerHooks(client)

	return client, nil
}

func registerHooks(client *ent.Client) {
	// 1. User Creation Hook: Auto-provision a default "vault" profile
	client.User.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			v, err := next.Mutate(ctx, m)
			if err != nil {
				return nil, err
			}
			if m.Op().Is(ent.OpCreate) {
				if userMut, ok := m.(*ent.UserMutation); ok {
					userID, ok := userMut.ID()
					if ok {
						var dbClient *ent.Client
						if tx, txErr := userMut.Tx(); txErr == nil {
							dbClient = tx.Client()
						} else {
							dbClient = client
						}

						err = dbClient.Profile.Create().
							SetUserID(userID).
							SetName("vault").
							SetWorkingDir("~/brain").
							SetIsDefault(true).
							Exec(ctx)
						if err != nil {
							return nil, fmt.Errorf("failed to auto-provision default vault profile: %w", err)
						}
					}
				}
			}
			return v, nil
		})
	})

	// 2. Profile default hook: enforce single active default profile per user
	client.Profile.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if !m.Op().Is(ent.OpCreate | ent.OpUpdate | ent.OpUpdateOne) {
				return next.Mutate(ctx, m)
			}
			profMut, ok := m.(*ent.ProfileMutation)
			if !ok {
				return next.Mutate(ctx, m)
			}

			isDefault, exists := profMut.IsDefault()
			if !exists || !isDefault {
				return next.Mutate(ctx, m)
			}

			userID, exists := profMut.UserID()

			v, err := next.Mutate(ctx, m)
			if err != nil {
				return nil, err
			}

			profileID, ok := profMut.ID()
			if !ok {
				return v, nil
			}

			if !exists {
				// Query user_id from DB if not in mutation
				var dbProfile *ent.Profile
				var dbErr error
				if tx, txErr := profMut.Tx(); txErr == nil {
					dbProfile, dbErr = tx.Client().Profile.Query().Where(profile.ID(profileID)).Only(ctx)
				} else {
					dbProfile, dbErr = client.Profile.Query().Where(profile.ID(profileID)).Only(ctx)
				}
				if dbErr == nil {
					userID = dbProfile.UserID
				}
			}

			if userID != uuid.Nil {
				var dbClient *ent.Client
				if tx, txErr := profMut.Tx(); txErr == nil {
					dbClient = tx.Client()
				} else {
					dbClient = client
				}

				_, err = dbClient.Profile.Update().
					Where(
						profile.UserID(userID),
						profile.IDNEQ(profileID),
					).
					SetIsDefault(false).
					Save(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to clear other default profiles: %w", err)
				}
			}
			return v, nil
		})
	})

	// 3. Message validation & AuditLog auto-write hook
	client.Message.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			msgMut, ok := m.(*ent.MessageMutation)
			if !ok {
				return next.Mutate(ctx, m)
			}

			// Validate "tool" role content
			if m.Op().Is(ent.OpCreate | ent.OpUpdate | ent.OpUpdateOne) {
				role, rExists := msgMut.Role()
				content, cExists := msgMut.Content()

				if !rExists || !cExists {
					if id, ok := msgMut.ID(); ok {
						var dbMsg *ent.Message
						var dbErr error
						if tx, txErr := msgMut.Tx(); txErr == nil {
							dbMsg, dbErr = tx.Client().Message.Query().Where(message.ID(id)).Only(ctx)
						} else {
							dbMsg, dbErr = client.Message.Query().Where(message.ID(id)).Only(ctx)
						}
						if dbErr == nil {
							if !rExists {
								role = dbMsg.Role
							}
							if !cExists {
								content = dbMsg.Content
							}
						}
					}
				}

				if role == message.RoleTool {
					var data map[string]interface{}
					if err := json.Unmarshal(content, &data); err != nil {
						return nil, fmt.Errorf("invalid json content for tool message: %w", err)
					}
					if _, ok := data["tool_call_id"]; !ok {
						return nil, fmt.Errorf("missing tool_call_id in tool message content")
					}
					if _, ok := data["result"]; !ok {
						return nil, fmt.Errorf("missing result in tool message content")
					}
				}
			}

			v, err := next.Mutate(ctx, m)
			if err != nil {
				return nil, err
			}

			msgID, ok := msgMut.ID()
			if !ok {
				return v, nil
			}

			var dbClient *ent.Client
			if tx, txErr := msgMut.Tx(); txErr == nil {
				dbClient = tx.Client()
			} else {
				dbClient = client
			}

			msg, errMsg := dbClient.Message.Query().
				Where(message.ID(msgID)).
				WithSession(func(q *ent.SessionQuery) {
					q.WithAccount()
				}).
				Only(ctx)

			if errMsg == nil && msg.Edges.Session != nil && msg.Edges.Session.Edges.Account != nil {
				userID := msg.Edges.Session.Edges.Account.UserID
				action := "message_mutation"
				if m.Op().Is(ent.OpCreate) {
					action = "message_create"
				} else if m.Op().Is(ent.OpUpdate | ent.OpUpdateOne) {
					action = "message_update"
				}

				metadataJSON, _ := json.Marshal(map[string]interface{}{
					"message_id": msgID,
					"session_id": msg.SessionID,
				})

				_, _ = dbClient.AuditLog.Create().
					SetUserID(userID).
					SetAction(action).
					SetEntityType("message").
					SetEntityID(msgID).
					SetMetadata(metadataJSON).
					Save(ctx)
			}

			return v, nil
		})
	})

	// 4. AuditLog immutability hook
	client.AuditLog.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if !m.Op().Is(ent.OpCreate) {
				return nil, fmt.Errorf("auditlog is immutable")
			}
			return next.Mutate(ctx, m)
		})
	})

	// 5. Global Soft-Delete Interceptor
	client.Intercept(
		ent.InterceptFunc(func(next ent.Querier) ent.Querier {
			return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
				if skip, _ := ctx.Value(mixin.SoftDeleteCtxKey{}).(bool); !skip {
					switch q := q.(type) {
					case *ent.UserQuery:
						q.Where(user.DeletedIsNil())
					case *ent.AccountQuery:
						q.Where(account.DeletedIsNil())
					case *ent.ProfileQuery:
						q.Where(profile.DeletedIsNil())
					case *ent.SessionQuery:
						q.Where(session.DeletedIsNil())
					case *ent.MessageQuery:
						q.Where(message.DeletedIsNil())
					}
				}
				return next.Query(ctx, q)
			})
		}),
	)
}
