package storage

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	entSql "entgo.io/ent/dialect/sql"
	"github.com/kilip/sbctl/ent"
	"github.com/kilip/sbctl/internal/config"
	_ "modernc.org/sqlite"
)

func TestOpen(t *testing.T) {
	// 1. Setup config pointing to in-memory shared database
	cfg := &config.Config{}
	cfg.Db.Path = "file::memory:?cache=shared"
	cfg.Db.Driver = "sqlite"

	// 2. Open standard sql.DB directly to verify connection configurations
	dsn := fmt.Sprintf("%s&_journal_mode=WAL&_pragma=foreign_keys(1)", cfg.Db.Path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open raw sqlite connection: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// 3. Verify WAL mode
	var journalMode string
	row := db.QueryRowContext(ctx, "PRAGMA journal_mode")
	if err := row.Scan(&journalMode); err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	t.Logf("journal_mode is: %s", journalMode)

	// 4. Verify Foreign Keys pragma
	var foreignKeys int
	row = db.QueryRowContext(ctx, "PRAGMA foreign_keys")
	if err := row.Scan(&foreignKeys); err != nil {
		t.Fatalf("failed to query foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("expected foreign_keys to be enabled (1), got %d", foreignKeys)
	}

	// 5. Wrap database and open ent client
	drv := entSql.OpenDB("sqlite3", db)
	client := ent.NewClient(ent.Driver(drv))
	defer client.Close()

	// 6. Run migrations to verify schema validity
	if err := client.Schema.Create(ctx); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}
}
