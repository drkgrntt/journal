package testutil

import (
	"journal/internal/models"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User inserts a user the way signup does (creator and updater are uuid.Nil).
// Pass mutate to adjust fields before the insert, e.g. to set a subscription
// status. The password is the plain text "password"; BeforeCreate hashes it.
func User(t testing.TB, tx *gorm.DB, email string, mutate ...func(*models.User)) *models.User {
	t.Helper()
	u := &models.User{
		Email:    email,
		Password: "password",
		Base:     &models.Base{CreatorID: &uuid.Nil, LastUpdaterID: &uuid.Nil},
	}
	for _, m := range mutate {
		m(u)
	}
	if err := tx.Create(u).Error; err != nil {
		t.Fatalf("create test user: %v", err)
	}
	return u
}
