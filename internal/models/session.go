package models

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func init() {
	registerModel(&Session{})
}

const (
	// SessionLifetime is how long a session lasts after its last rotation, so
	// someone who keeps using the app stays signed in.
	SessionLifetime = 30 * 24 * time.Hour
	// SessionRotateAfter is how often the cookie's token is replaced.
	SessionRotateAfter = 24 * time.Hour
	// SessionOverlap is how long the previous token still works after a
	// rotation, so requests already in flight (parallel HTMX calls, a lost
	// response) don't sign the person out.
	SessionOverlap = 5 * time.Minute
)

// Session is one signed-in browser. The cookie carries a random token; only its
// hash is stored. It deliberately records no IP address or user agent.
type Session struct {
	ID                 uuid.UUID  `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	UserID             uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash          string     `gorm:"type:text;not null;uniqueIndex"`
	PreviousTokenHash  *string    `gorm:"type:text;index"`
	PreviousValidUntil *time.Time `gorm:"type:timestamptz"`
	CreatedAt          time.Time  `gorm:"type:timestamptz;not null"`
	RotatedAt          time.Time  `gorm:"type:timestamptz;not null"`
	ExpiresAt          time.Time  `gorm:"type:timestamptz;not null;index"`
}

// HashSessionToken is what the database stores for a cookie token.
func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// newSessionToken returns a random token (256 bits) and its hash.
func newSessionToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashSessionToken(token), nil
}

// CreateSession signs a user in and returns the token for their cookie.
func CreateSession(db *gorm.DB, userID uuid.UUID, now time.Time) (string, error) {
	token, hash, err := newSessionToken()
	if err != nil {
		return "", err
	}
	s := &Session{
		UserID:    userID,
		TokenHash: hash,
		CreatedAt: now,
		RotatedAt: now,
		ExpiresAt: now.Add(SessionLifetime),
	}
	if err := db.Create(s).Error; err != nil {
		return "", err
	}
	return token, nil
}

// FindSession returns the live session a cookie token belongs to. The token
// may be the current one or, briefly after a rotation, the previous one.
func FindSession(db *gorm.DB, token string, now time.Time) (*Session, error) {
	if token == "" {
		return nil, gorm.ErrRecordNotFound
	}
	hash := HashSessionToken(token)
	var s Session
	err := db.
		Where("expires_at > ?", now).
		Where("token_hash = ? OR (previous_token_hash = ? AND previous_valid_until > ?)", hash, hash, now).
		First(&s).Error
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// NeedsRotation reports whether the cookie should get a fresh token. Only the
// current token triggers it; the previous one is just riding out its overlap.
func (s *Session) NeedsRotation(token string, now time.Time) bool {
	return s.TokenHash == HashSessionToken(token) && now.Sub(s.RotatedAt) >= SessionRotateAfter
}

// RotateSession swaps in a new token and extends the session. It returns
// ("", false) without error if another request rotated first; the caller's
// request is still valid and its cookie is simply left alone.
func RotateSession(db *gorm.DB, s *Session, now time.Time) (string, bool, error) {
	token, hash, err := newSessionToken()
	if err != nil {
		return "", false, err
	}
	until := now.Add(SessionOverlap)
	res := db.Model(&Session{}).
		Where("id = ? AND token_hash = ?", s.ID, s.TokenHash).
		Updates(map[string]any{
			"token_hash":           hash,
			"previous_token_hash":  s.TokenHash,
			"previous_valid_until": until,
			"rotated_at":           now,
			"expires_at":           now.Add(SessionLifetime),
		})
	if res.Error != nil {
		return "", false, res.Error
	}
	if res.RowsAffected == 0 {
		return "", false, nil
	}
	return token, true, nil
}

// DeleteSessionByToken signs out the browser holding this token.
func DeleteSessionByToken(db *gorm.DB, token string) error {
	if token == "" {
		return nil
	}
	hash := HashSessionToken(token)
	return db.Where("token_hash = ? OR previous_token_hash = ?", hash, hash).Delete(&Session{}).Error
}

// DeleteUserSessions signs a user out everywhere, except the session keep
// names (pass nil to end them all).
func DeleteUserSessions(db *gorm.DB, userID uuid.UUID, keep *uuid.UUID) error {
	q := db.Where("user_id = ?", userID)
	if keep != nil {
		q = q.Where("id <> ?", *keep)
	}
	return q.Delete(&Session{}).Error
}

// DeleteExpiredSessions removes sessions that can no longer be used.
func DeleteExpiredSessions(db *gorm.DB, now time.Time) (int64, error) {
	res := db.Where("expires_at <= ?", now).Delete(&Session{})
	return res.RowsAffected, res.Error
}
