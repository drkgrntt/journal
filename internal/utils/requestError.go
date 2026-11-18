package utils

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// SetRequestError records why a request is about to fail, for the server-side
// error log line that carries the request's reference. Call it right before
// returning a 5xx. Only a sanitized description is kept (see ErrorCause).
func SetRequestError(ctx *fiber.Ctx, err error) {
	if err == nil {
		return
	}
	cause := ErrorCause(err)
	ctx.Locals(LocalRequestError, &cause)
}

// ErrorCause describes err without risking anything the person wrote. Database
// error messages often quote the offending values (an email in a unique
// violation, say), and other errors can echo input, so the full text is never
// logged: known-safe sentinel errors by message, Postgres errors by code,
// table and constraint only, and anything else by Go type alone.
func ErrorCause(err error) string {
	if err == nil {
		return ""
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fmt.Sprintf("postgres error %s (table=%s constraint=%s)", pgErr.Code, pgErr.TableName, pgErr.ConstraintName)
	}
	for _, safe := range []error{
		gorm.ErrRecordNotFound,
		context.Canceled,
		context.DeadlineExceeded,
	} {
		if errors.Is(err, safe) {
			return safe.Error()
		}
	}
	return fmt.Sprintf("%T", err)
}
