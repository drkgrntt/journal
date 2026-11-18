// Package testutil holds helpers for tests that need a real database.
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"journal/internal/database"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Main runs a package's tests against a migrated Postgres and returns the exit
// code for os.Exit. Use it from TestMain:
//
//	func TestMain(m *testing.M) { os.Exit(testutil.Main(m)) }
//
// The database is a throwaway container, or the one named by TEST_DATABASE_URL
// when set (never point that at a database with data you care about: the
// schema is created in it as-is and tests leave rows behind unless they use
// Tx). Tests then call DB or Tx.
func Main(m *testing.M) int {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		ctx := context.Background()
		c, err := tcpostgres.Run(ctx, "postgres:16-alpine",
			tcpostgres.WithDatabase("journal_test"),
			tcpostgres.WithUsername("test"),
			tcpostgres.WithPassword("test"),
			testcontainers.WithWaitStrategy(
				wait.ForLog("database system is ready to accept connections").
					WithOccurrence(2).
					WithStartupTimeout(60*time.Second)),
		)
		if err != nil {
			fmt.Fprintln(os.Stderr, "could not start postgres container (is Docker running? or set TEST_DATABASE_URL):", err)
			return 1
		}
		defer c.Terminate(ctx)
		if url, err = c.ConnectionString(ctx, "sslmode=disable"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	var err error
	if testDB, err = open(url); err != nil {
		fmt.Fprintln(os.Stderr, "could not open test database:", err)
		return 1
	}
	return m.Run()
}

var testDB *gorm.DB

func open(url string) (*gorm.DB, error) {
	time.Local = time.UTC
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Error),
		DisableForeignKeyConstraintWhenMigrating: true,
		NowFunc:                                  func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := applyAll(sqlDB); err != nil {
		return nil, err
	}
	return db, nil
}

func applyAll(sqlDB *sql.DB) error { return database.ApplyMigrations(sqlDB) }

// DB returns the shared migrated test database. Prefer Tx for isolation.
func DB(t testing.TB) *gorm.DB {
	t.Helper()
	if testDB == nil {
		t.Fatal("testutil.Main was not used in TestMain")
	}
	return testDB
}

type rollback struct{}

func (rollback) Error() string { return "rollback" }

// Tx runs fn inside a transaction that is always rolled back, so each test
// starts from a clean slate and leaves nothing behind. Code under test that
// opens its own transaction runs as a savepoint (GORM nests them). Code that
// takes a *gorm.DB can be handed tx. Code that uses a package-level DB (the
// middleware and jobs packages) can be pointed at it with their SetDB.
func Tx(t testing.TB, fn func(tx *gorm.DB)) {
	t.Helper()
	err := DB(t).Transaction(func(tx *gorm.DB) error {
		fn(tx)
		return rollback{}
	})
	if err != nil && err != (rollback{}) {
		t.Fatal(err)
	}
}
