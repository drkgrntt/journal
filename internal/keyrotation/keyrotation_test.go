package keyrotation

import (
	"strings"
	"testing"

	"journal/internal/testutil"
	"journal/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	krOldKey = "0123456789abcdef0123456789abcdef"
	krNewKey = "ZYXWVUTSRQPONMLKJIHGFEDCBA987654"
	krSecret = "private words that must never leak"
)

func krEnc(t *testing.T, key, s string) string {
	t.Helper()
	c, err := utils.EncryptWithKey([]byte(key), s)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func krDec(t *testing.T, key, c string) (string, error) {
	t.Helper()
	return utils.DecryptWithKey([]byte(key), c)
}

// krSeed inserts one row in every target table, encrypted with krOldKey, and
// returns the (table, column) -> id map.
func krSeed(t *testing.T, tx *gorm.DB) map[string]string {
	t.Helper()
	ids := map[string]string{}
	uid := uuid.New().String()
	ins := func(table string, args ...any) {
		id := uuid.New().String()
		ids[table] = id
		if err := tx.Exec(args[0].(string), append([]any{id, uid, uid}, args[1:]...)...).Error; err != nil {
			t.Fatalf("seed %s: %v", table, err)
		}
	}
	e := krEnc(t, krOldKey, krSecret)
	ins("journals", "INSERT INTO journals (id, creator_id, last_updater_id, date, entry, journal_type_id, rating_id, is_encrypted) VALUES (?, ?, ?, now(), ?, 1, 1, true)", e)
	ins("action_items", "INSERT INTO action_items (id, creator_id, last_updater_id, text, is_encrypted) VALUES (?, ?, ?, ?, true)", e)
	ins("thankfuls", "INSERT INTO thankfuls (id, creator_id, last_updater_id, text, is_encrypted) VALUES (?, ?, ?, ?, true)", e)
	ins("recurring_action_items", "INSERT INTO recurring_action_items (id, creator_id, last_updater_id, text, is_encrypted) VALUES (?, ?, ?, ?, true)", e)
	ins("reminders", "INSERT INTO reminders (id, creator_id, last_updater_id, text, is_encrypted) VALUES (?, ?, ?, ?, true)", e)
	ins("entry_templates", "INSERT INTO entry_templates (id, creator_id, last_updater_id, name, body, is_encrypted) VALUES (?, ?, ?, ?, ?, true)", e, krEnc(t, krOldKey, "body "+krSecret))

	u := testutil.User(t, tx, "kr-"+uuid.NewString()+"@example.com")
	if err := tx.Exec("UPDATE users SET reminder_email_message = ? WHERE id = ?", e, u.ID).Error; err != nil {
		t.Fatal(err)
	}
	ids["users"] = u.ID.String()
	return ids
}

func krCell(t *testing.T, tx *gorm.DB, table, col, id string) string {
	t.Helper()
	var v string
	if err := tx.Raw("SELECT "+col+" FROM "+table+" WHERE id::text = ?", id).Scan(&v).Error; err != nil {
		t.Fatal(err)
	}
	return v
}

func TestRotateReEncryptsEveryTable(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		ids := krSeed(t, tx)
		rep, err := Rotate(tx, []byte(krOldKey), []byte(krNewKey), true)
		if err != nil {
			t.Fatal(err)
		}
		if rep.Total() < 7 {
			t.Errorf("encrypted rows = %d, want at least 7", rep.Total())
		}
		for _, tg := range Targets {
			for _, col := range tg.Columns {
				c := krCell(t, tx, tg.Table, col, ids[tg.Table])
				got, err := krDec(t, krNewKey, c)
				if err != nil || !strings.Contains(got, krSecret) {
					t.Errorf("%s.%s: new key gives %q, %v", tg.Table, col, got, err)
				}
				if _, err := krDec(t, krOldKey, c); err == nil {
					t.Errorf("%s.%s: old key still decrypts", tg.Table, col)
				}
			}
		}
	})
}

func TestDryRunChangesNothing(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		ids := krSeed(t, tx)
		before := krCell(t, tx, "journals", "entry", ids["journals"])
		rep, err := Rotate(tx, []byte(krOldKey), []byte(krNewKey), false)
		if err != nil {
			t.Fatal(err)
		}
		if rep.Applied || rep.Total() < 7 {
			t.Errorf("report = %+v, want a counted dry run", rep)
		}
		if after := krCell(t, tx, "journals", "entry", ids["journals"]); after != before {
			t.Error("dry run changed a row")
		}
	})
}

func TestWrongOldKeyChangesNothingAndLeaksNothing(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		ids := krSeed(t, tx)
		before := krCell(t, tx, "users", "reminder_email_message", ids["users"])
		wrong := "wrongwrongwrongwrongwrongwrongwr"
		_, err := Rotate(tx, []byte(wrong), []byte(krNewKey), true)
		if err == nil {
			t.Fatal("expected an error")
		}
		msg := err.Error()
		if strings.Contains(msg, krSecret) || strings.Contains(msg, before) {
			t.Errorf("error leaks data: %q", msg)
		}
		if !strings.Contains(msg, "row ") {
			t.Errorf("error should name the row: %q", msg)
		}
		if after := krCell(t, tx, "users", "reminder_email_message", ids["users"]); after != before {
			t.Error("a failed rotation changed a row")
		}
		if _, err := krDec(t, krOldKey, krCell(t, tx, "journals", "entry", ids["journals"])); err != nil {
			t.Error("journal no longer decrypts with the old key after a failed rotation")
		}
	})
}

func TestPlaintextAndEmptyRowsAreLeftAlone(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		uid := uuid.New().String()
		id := uuid.New().String()
		if err := tx.Exec("INSERT INTO thankfuls (id, creator_id, last_updater_id, text, is_encrypted) VALUES (?, ?, ?, 'plain old text', false)", id, uid, uid).Error; err != nil {
			t.Fatal(err)
		}
		u := testutil.User(t, tx, "kr-"+uuid.NewString()+"@example.com") // no reminder message
		if _, err := Rotate(tx, []byte(krOldKey), []byte(krNewKey), true); err != nil {
			t.Fatal(err)
		}
		if got := krCell(t, tx, "thankfuls", "text", id); got != "plain old text" {
			t.Errorf("plaintext row changed to %q", got)
		}
		var n int
		tx.Raw("SELECT count(*) FROM users WHERE id = ? AND reminder_email_message IS NULL", u.ID).Scan(&n)
		if n != 1 {
			t.Error("a user with no message was given one")
		}
	})
}

func TestRotationPagesThroughManyRows(t *testing.T) {
	old := batchSize
	batchSize = 2
	defer func() { batchSize = old }()
	testutil.Tx(t, func(tx *gorm.DB) {
		uid := uuid.New().String()
		var ids []string
		for i := 0; i < 7; i++ {
			id := uuid.New().String()
			ids = append(ids, id)
			if err := tx.Exec("INSERT INTO reminders (id, creator_id, last_updater_id, text, is_encrypted) VALUES (?, ?, ?, ?, true)", id, uid, uid, krEnc(t, krOldKey, krSecret)).Error; err != nil {
				t.Fatal(err)
			}
		}
		if _, err := Rotate(tx, []byte(krOldKey), []byte(krNewKey), true); err != nil {
			t.Fatal(err)
		}
		for _, id := range ids {
			if _, err := krDec(t, krNewKey, krCell(t, tx, "reminders", "text", id)); err != nil {
				t.Errorf("row %s was skipped by paging: %v", id, err)
			}
		}
	})
}

func TestCheck(t *testing.T) {
	testutil.Tx(t, func(tx *gorm.DB) {
		krSeed(t, tx)
		if _, err := Check(tx, []byte(krOldKey)); err != nil {
			t.Errorf("right key: %v", err)
		}
		if _, err := Check(tx, []byte(krNewKey)); err == nil {
			t.Error("wrong key passed the check")
		}
		if _, err := Check(tx, []byte("short")); err == nil {
			t.Error("short key accepted")
		}
	})
}

func TestValidateKeys(t *testing.T) {
	a, b := []byte(krOldKey), []byte(krNewKey)
	for name, tc := range map[string]struct{ o, n []byte }{
		"old short": {[]byte("x"), b}, "new short": {a, []byte("x")}, "same": {a, a}, "empty new": {a, nil},
	} {
		if ValidateKeys(tc.o, tc.n) == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if err := ValidateKeys(a, b); err != nil {
		t.Errorf("valid pair rejected: %v", err)
	}
}

func TestGenerateKey(t *testing.T) {
	a, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := GenerateKey()
	if len(a) != KeyLength || a == b {
		t.Errorf("keys %q %q: want two distinct %d-char keys", a, b, KeyLength)
	}
	for _, r := range a {
		if !strings.ContainsRune(keyAlphabet, r) {
			t.Errorf("unexpected character %q (must be safe unquoted in .env)", r)
		}
	}
}

// Fails when a table gains an is_encrypted column (or users gains another
// encrypted column) that Targets doesn't list, which would leave data
// unreadable after a rotation.
func TestTargetsCoverEveryEncryptedTable(t *testing.T) {
	var tables []string
	if err := testutil.DB(t).Raw("SELECT table_name FROM information_schema.columns WHERE table_schema = 'public' AND column_name = 'is_encrypted'").Scan(&tables).Error; err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, tg := range Targets {
		listed[tg.Table] = true
	}
	for _, tb := range tables {
		if !listed[tb] {
			t.Errorf("table %s has is_encrypted but is not in Targets", tb)
		}
	}
	for _, tg := range Targets {
		if tg.Flag == "is_encrypted" {
			found := false
			for _, tb := range tables {
				found = found || tb == tg.Table
			}
			if !found {
				t.Errorf("Targets lists %s but it has no is_encrypted column", tg.Table)
			}
		}
	}
}
