package models

import (
	"time"
)

// Derived from MD files, not the DB
type Blog struct {
	Title       string
	Slug        string
	Date        *time.Time
	Description string
	Content     string
}
