package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const createMigrationsTableSQL = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version TEXT PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`

// RunMigrations applies any pending hand-written SQL migrations from
// internal/database/migrations, in order, each in its own transaction,
// recording applied versions in a schema_migrations table. These cover
// schema changes AutoMigrate can't safely express against a database that
// already has real data - dropping/replacing constraints, retyping
// columns, backfills, etc. Call this AFTER AutoMigrate: AutoMigrate creates
// tables/columns/indexes from the current model tags (so a fresh database
// gets the correct schema directly, and these migrations become no-ops on
// it), while this step fixes up whatever AutoMigrate can't retroactively
// change on a database that already exists in an older, incorrect state.
func RunMigrations() {
	sqlDB, err := dbInstance.DB.DB()
	if err != nil {
		log.Fatal(err)
	}

	if _, err := sqlDB.Exec(createMigrationsTableSQL); err != nil {
		log.Fatal(err)
	}

	versions, err := pendingMigrationVersions(sqlDB)
	if err != nil {
		log.Fatal(err)
	}

	for _, version := range versions {
		if err := applyMigration(sqlDB, version); err != nil {
			log.Fatal(err)
		}
	}
}

// pendingMigrationVersions returns the sorted list of migration versions
// (derived from *.up.sql filenames in internal/database/migrations) that
// aren't yet recorded in schema_migrations.
func pendingMigrationVersions(db *sql.DB) ([]string, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	var allVersions []string
	for _, entry := range entries {
		if version, ok := strings.CutSuffix(entry.Name(), ".up.sql"); ok {
			allVersions = append(allVersions, version)
		}
	}
	sort.Strings(allVersions)

	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var pending []string
	for _, version := range allVersions {
		if !applied[version] {
			pending = append(pending, version)
		}
	}
	return pending, nil
}

func applyMigration(db *sql.DB, version string) error {
	sqlBytes, err := migrationsFS.ReadFile("migrations/" + version + ".up.sql")
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("migration %s failed: %w", version, err)
	}

	if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("applied migration %s", version)
	return nil
}
