// Package keyrotation re-encrypts everything the app stores encrypted
// (journal entries, action items, thankfuls, routines, reminders, entry
// templates, and the reminder-email message) from one CRYPTO_KEY to another.
//
// It works on the raw columns with SQL, never through the models, so their
// hooks can't run. It never logs or returns plaintext or ciphertext: errors name
// a table and row id only.
package keyrotation

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"journal/internal/utils"

	"gorm.io/gorm"
)

// KeyLength is the size of the AES-256 key the app uses.
const KeyLength = 32

// Target is one table with encrypted text columns. Flag names the boolean
// column that says a row is encrypted ("" means a non-empty value is always
// ciphertext, as for users.reminder_email_message).
type Target struct {
	Table   string
	Columns []string
	Flag    string
}

// Targets lists every encrypted column in the schema. Add a row here when a
// model starts encrypting something new (TestTargetsCoverEveryEncryptedModel
// fails if one is missed).
var Targets = []Target{
	{"journals", []string{"entry"}, "is_encrypted"},
	{"action_items", []string{"text"}, "is_encrypted"},
	{"thankfuls", []string{"text"}, "is_encrypted"},
	{"recurring_action_items", []string{"text"}, "is_encrypted"},
	{"reminders", []string{"text"}, "is_encrypted"},
	{"entry_templates", []string{"name", "body"}, "is_encrypted"},
	{"users", []string{"reminder_email_message"}, ""},
}

// batchSize is how many rows are read at a time. A variable so tests can make
// it small enough to exercise paging.
var batchSize = 500

// TableReport counts what happened in one table.
type TableReport struct {
	Table     string
	Rows      int // rows examined
	Encrypted int // rows whose columns were decrypted (and re-encrypted, unless a check)
	Plaintext int // rows flagged as not encrypted, left alone
	Empty     int // rows with nothing to encrypt
}

// Report is the result of a Check or Rotate.
type Report struct {
	Tables  []TableReport
	Applied bool
}

// Total returns the number of rows that were encrypted.
func (r Report) Total() int {
	n := 0
	for _, t := range r.Tables {
		n += t.Encrypted
	}
	return n
}

// ValidateKeys checks a pair of keys for rotation.
func ValidateKeys(oldKey, newKey []byte) error {
	if len(oldKey) != KeyLength {
		return fmt.Errorf("the current key must be exactly %d bytes (it is %d)", KeyLength, len(oldKey))
	}
	if len(newKey) != KeyLength {
		return fmt.Errorf("the new key must be exactly %d bytes (it is %d)", KeyLength, len(newKey))
	}
	if string(oldKey) == string(newKey) {
		return errors.New("the new key is the same as the current key")
	}
	return nil
}

const keyAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// GenerateKey returns a random 32-character key using only letters and digits.
// No shell- or dotenv-special characters, so it survives being pasted into .env
// unquoted (a key with %, ^ or & did not).
func GenerateKey() (string, error) {
	b := make([]byte, KeyLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(keyAlphabet))))
		if err != nil {
			return "", err
		}
		b[i] = keyAlphabet[n.Int64()]
	}
	return string(b), nil
}

type row struct {
	id    string
	flag  bool
	cells []*string
}

// readBatch returns up to batchSize rows with id greater than after, in id order.
func readBatch(db *gorm.DB, t Target, after string) ([]row, error) {
	cols := "id::text"
	for _, c := range t.Columns {
		cols += ", " + c
	}
	if t.Flag != "" {
		cols += ", " + t.Flag
	}
	q := fmt.Sprintf("SELECT %s FROM %s WHERE id::text > ? ORDER BY id::text LIMIT %d", cols, t.Table, batchSize)
	rows, err := db.Raw(q, after).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []row
	for rows.Next() {
		r := row{cells: make([]*string, len(t.Columns))}
		dest := []any{&r.id}
		for i := range r.cells {
			dest = append(dest, &r.cells[i])
		}
		var flag *bool
		if t.Flag != "" {
			dest = append(dest, &flag)
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		r.flag = t.Flag == "" || (flag != nil && *flag)
		out = append(out, r)
	}
	return out, rows.Err()
}

// transform runs fn over every encrypted cell and, when fn returns a new value,
// writes it. It returns what it saw.
func walk(db *gorm.DB, t Target, fn func(cell string) (string, error), write bool) (TableReport, error) {
	rep := TableReport{Table: t.Table}
	after := ""
	for {
		batch, err := readBatch(db, t, after)
		if err != nil {
			return rep, fmt.Errorf("%s: reading rows: %w", t.Table, err)
		}
		if len(batch) == 0 {
			return rep, nil
		}
		for _, r := range batch {
			after = r.id
			rep.Rows++

			if !r.flag {
				rep.Plaintext++
				continue
			}
			updates := map[string]any{}
			for i, cell := range r.cells {
				if cell == nil || *cell == "" {
					continue
				}
				out, err := fn(*cell)
				if err != nil {
					// Row id only: never the value, never the underlying text.
					return rep, fmt.Errorf("%s row %s column %s: %w", t.Table, r.id, t.Columns[i], err)
				}
				updates[t.Columns[i]] = out
			}
			if len(updates) == 0 {
				rep.Empty++
				continue
			}
			rep.Encrypted++
			if write {
				if err := db.Table(t.Table).Where("id::text = ?", r.id).UpdateColumns(updates).Error; err != nil {
					return rep, fmt.Errorf("%s row %s: writing: %w", t.Table, r.id, err)
				}
			}
		}
	}
}

// Check decrypts every encrypted value with key and changes nothing. Use it
// before a rotation (with the current key) and after (with the new one).
func Check(db *gorm.DB, key []byte) (Report, error) {
	if len(key) != KeyLength {
		return Report{}, fmt.Errorf("the key must be exactly %d bytes (it is %d)", KeyLength, len(key))
	}
	var rep Report
	for _, t := range Targets {
		tr, err := walk(db, t, func(cell string) (string, error) {
			if _, err := utils.DecryptWithKey(key, cell); err != nil {
				return "", errors.New("does not decrypt with this key")
			}
			return cell, nil
		}, false)
		rep.Tables = append(rep.Tables, tr)
		if err != nil {
			return rep, err
		}
	}
	return rep, nil
}

// Rotate re-encrypts every encrypted value from oldKey to newKey. Everything
// runs in one transaction: if any value fails to decrypt with oldKey, or does
// not decrypt back to the same text with newKey, nothing is changed. With apply
// false it does all the same work and then rolls back, so a dry run proves the
// real run will succeed.
//
// The app must be stopped first: a write made with the old key during or after
// the rotation can't be read with the new one.
func Rotate(db *gorm.DB, oldKey, newKey []byte, apply bool) (Report, error) {
	if err := ValidateKeys(oldKey, newKey); err != nil {
		return Report{}, err
	}
	var rep Report
	rep.Applied = apply
	errDryRun := errors.New("dry run")

	err := db.Transaction(func(tx *gorm.DB) error {
		for _, t := range Targets {
			// Block writers to this table until we commit.
			if err := tx.Exec("LOCK TABLE " + t.Table + " IN EXCLUSIVE MODE").Error; err != nil {
				return fmt.Errorf("%s: locking: %w", t.Table, err)
			}
			tr, err := walk(tx, t, func(cell string) (string, error) {
				plain, err := utils.DecryptWithKey(oldKey, cell)
				if err != nil {
					return "", errors.New("does not decrypt with the current key")
				}
				fresh, err := utils.EncryptWithKey(newKey, plain)
				if err != nil {
					return "", errors.New("could not encrypt with the new key")
				}
				back, err := utils.DecryptWithKey(newKey, fresh)
				if err != nil || back != plain {
					return "", errors.New("did not read back the same after re-encrypting")
				}
				return fresh, nil
			}, apply)
			rep.Tables = append(rep.Tables, tr)
			if err != nil {
				return err
			}
		}
		if !apply {
			return errDryRun // roll back
		}
		return nil
	})
	if err != nil && !errors.Is(err, errDryRun) {
		rep.Applied = false
		return rep, err
	}
	return rep, nil
}
