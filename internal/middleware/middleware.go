package middleware

import (
	"journal/internal/database"
	"sync"

	"gorm.io/gorm"
)

var (
	db     *gorm.DB
	dbOnce sync.Once
)

// DB returns the database the middleware uses, connecting on first use rather
// than at import, so importing this package (as controllers does) doesn't need
// a live Postgres. Tests call SetDB to point it at a test database.
func DB() *gorm.DB {
	dbOnce.Do(func() {
		if db == nil {
			db = database.New().DB
		}
	})
	return db
}

// SetDB overrides the database. Call it before any request is served.
func SetDB(d *gorm.DB) {
	db = d
}
