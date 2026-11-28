package controllers

import (
	"journal/internal/models"
	"journal/internal/testutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"gorm.io/gorm"
)

// Deleting something removes the row, not just marks it deleted.
func TestDeleteRemovesTheRow(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	for _, c := range authzCases() {
		if c.method != http.MethodDelete {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			testutil.Tx(t, func(tx *gorm.DB) {
				owner := compedUser(t, tx, "owner@example.com")
				id := c.create(t, tx, owner)
				app := newAuthedTestApp(t, tx, owner, c.controller())

				code, body := send(t, app, http.MethodDelete, strings.Replace(c.path, "%s", id, 1), nil)
				if !is2xx(code) {
					t.Fatalf("status %d: %s", code, body)
				}
				if row := rawRow(t, tx, c.table, id); row != "" {
					t.Errorf("row is still stored: %s", row)
				}
			})
		})
	}
}

// Migration 000015 removes what earlier soft deletes left behind, and only that.
func TestPurgeSoftDeletedMigration(t *testing.T) {
	t.Setenv("CRYPTO_KEY", testCryptoKey)
	sql, err := os.ReadFile("../database/migrations/000015_purge_soft_deleted.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	testutil.Tx(t, func(tx *gorm.DB) {
		user := compedUser(t, tx, "purge@example.com")
		var checked int
		for _, c := range authzCases() {
			if c.method != http.MethodDelete {
				continue
			}
			checked++
			gone := c.create(t, tx, user)
			// custom topic names are unique, so free the name first
			if err := tx.Exec("UPDATE "+c.table+" SET deleted_at = now() WHERE id = ?", gone).Error; err != nil {
				t.Fatal(err)
			}
			kept := c.create(t, tx, user)

			if err := tx.Exec(string(sql)).Error; err != nil {
				t.Fatal(err)
			}
			if rawRow(t, tx, c.table, gone) != "" {
				t.Errorf("%s: soft deleted row survived", c.table)
			}
			if rawRow(t, tx, c.table, kept) == "" {
				t.Errorf("%s: live row was removed", c.table)
			}
		}
		if checked != 7 {
			t.Errorf("checked %d tables, want 7", checked)
		}
		var accounts int64
		tx.Model(&models.User{}).Where("id = ?", user.ID).Count(&accounts)
		if accounts != 1 {
			t.Error("the account was removed")
		}
	})
}
