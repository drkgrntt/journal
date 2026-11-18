package testutil

import (
	"journal/internal/models"
	"os"
	"testing"

	"gorm.io/gorm"
)

func TestMain(m *testing.M) { os.Exit(Main(m)) }

func TestTxRollsBack(t *testing.T) {
	var id string
	Tx(t, func(tx *gorm.DB) {
		id = User(t, tx, "rollback@example.com").ID.String()
	})
	var n int64
	DB(t).Model(&models.User{}).Where("id = ?", id).Count(&n)
	if n != 0 {
		t.Fatalf("user survived rollback: %d", n)
	}
}
