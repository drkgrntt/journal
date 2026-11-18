package utils

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type customErr struct{ secret string }

func (e *customErr) Error() string { return "failed for " + e.secret }

func TestErrorCause(t *testing.T) {
	pg := &pgconn.PgError{
		Code:           "23505",
		TableName:      "users",
		ConstraintName: "idx_users_email",
		Message:        "duplicate key value violates unique constraint",
		Detail:         "Key (email)=(someone@example.com) already exists.",
	}
	tests := []struct {
		name    string
		err     error
		want    string
		notWant []string
	}{
		{"nil", nil, "", nil},
		{"postgres error: code, table, constraint only", pg,
			"postgres error 23505 (table=users constraint=idx_users_email)", []string{"someone@example.com", "Key ("}},
		{"postgres error wrapped by fmt", fmt.Errorf("saving: %w", pg),
			"postgres error 23505 (table=users constraint=idx_users_email)", []string{"someone@example.com"}},
		{"record not found", fmt.Errorf("loading: %w", gorm.ErrRecordNotFound), gorm.ErrRecordNotFound.Error(), nil},
		{"context canceled", context.Canceled, "context canceled", nil},
		{"unknown error: type only, never the text", &customErr{secret: "my private entry text"},
			"*utils.customErr", []string{"my private entry text"}},
		{"plain errors.New: type only", errors.New("contains the user's words"),
			"*errors.errorString", []string{"user's words"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ErrorCause(tt.err)
			if got != tt.want {
				t.Fatalf("ErrorCause = %q, want %q", got, tt.want)
			}
			for _, bad := range tt.notWant {
				if strings.Contains(got, bad) {
					t.Fatalf("ErrorCause leaked %q: %q", bad, got)
				}
			}
		})
	}
}

func TestSetRequestError(t *testing.T) {
	runHandler(t, "/", func(ctx *fiber.Ctx) error {
		SetRequestError(ctx, nil)
		if GetLocal[string](ctx, LocalRequestError) != nil {
			t.Error("nil error should record nothing")
		}
		SetRequestError(ctx, &customErr{secret: "private"})
		got := GetLocal[string](ctx, LocalRequestError)
		if got == nil || *got != "*utils.customErr" {
			t.Errorf("recorded %v", got)
		}
		return ctx.SendString("ok")
	})
}
