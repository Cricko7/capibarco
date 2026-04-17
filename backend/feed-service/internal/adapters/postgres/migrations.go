package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type migration struct {
	Version int
	Name    string
	SQL     string
}

func (s *Store) applyMigrations(ctx context.Context) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin postgres migrations: %w", err)
	}
	defer rollback(tx)

	for _, migration := range migrations {
		applied, err := migrationApplied(ctx, tx, migration.Version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
			return fmt.Errorf("apply postgres migration %s: %w", migration.Name, err)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO feed_schema_migrations (version, name)
VALUES ($1, $2)
ON CONFLICT (version) DO NOTHING`, migration.Version, migration.Name); err != nil {
			return fmt.Errorf("record postgres migration %s: %w", migration.Name, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit postgres migrations: %w", err)
	}
	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("read postgres migrations: %w", err)
	}
	migrations := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := migrationVersion(entry.Name())
		if err != nil {
			return nil, err
		}
		raw, err := migrationFS.ReadFile(path.Join("migrations", entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read postgres migration %s: %w", entry.Name(), err)
		}
		migrations = append(migrations, migration{
			Version: version,
			Name:    entry.Name(),
			SQL:     string(raw),
		})
	}
	sort.Slice(migrations, func(i int, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

func migrationVersion(name string) (int, error) {
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("postgres migration %q must start with numeric version", name)
	}
	version, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, fmt.Errorf("parse postgres migration version %q: %w", name, err)
	}
	return version, nil
}

func migrationApplied(ctx context.Context, tx *sql.Tx, version int) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM information_schema.tables
	WHERE table_schema = current_schema()
		AND table_name = 'feed_schema_migrations'
)`).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check postgres migrations table: %w", err)
	}
	if !exists {
		return false, nil
	}
	err = tx.QueryRowContext(ctx, `
SELECT EXISTS (
	SELECT 1
	FROM feed_schema_migrations
	WHERE version = $1
)`, version).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check postgres migration version %d: %w", version, err)
	}
	return exists, nil
}
