package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type testKeys struct {
	priv *rsa.PrivateKey
	// base64-encoded PEM, as stored in the environment
	privEnv, pubEnv string
}

func newTestKeys(t *testing.T) testKeys {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	return testKeys{
		priv:    priv,
		privEnv: base64.StdEncoding.EncodeToString(privPEM),
		pubEnv:  base64.StdEncoding.EncodeToString(pubPEM),
	}
}

func setTokenEnv(t *testing.T, k testKeys, _ string) {
	t.Helper()
	t.Setenv("ACCESS_TOKEN_PRIVATE_KEY", k.privEnv)
	t.Setenv("ACCESS_TOKEN_PUBLIC_KEY", k.pubEnv)
}

func signWith(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestValidateTokenBadEnv(t *testing.T) {
	keys := newTestKeys(t)
	setTokenEnv(t, keys, "60m")
	tok, err := CreateResetToken(uuid.New(), 0)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		pub  string
	}{
		{"missing", ""},
		{"not base64", "%%%"},
		{"not PEM", base64.StdEncoding.EncodeToString([]byte("nope"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ACCESS_TOKEN_PUBLIC_KEY", tt.pub)
			if _, _, err := ValidateResetToken(tok); err == nil {
				t.Error("ValidateResetToken: expected error")
			}
			if _, _, err := ValidateEmailConfirmToken(tok); err == nil {
				t.Error("ValidateEmailConfirmToken: expected error")
			}
		})
	}
}

func TestEmailConfirmTokenRoundTrip(t *testing.T) {
	setTokenEnv(t, newTestKeys(t), "60m")
	id := uuid.New()
	tok, err := CreateEmailConfirmToken(id, "Some.One@Example.COM")
	if err != nil {
		t.Fatal(err)
	}
	gotID, email, err := ValidateEmailConfirmToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if gotID != id {
		t.Errorf("id = %s, want %s", gotID, id)
	}
	if email != "some.one@example.com" {
		t.Errorf("email = %q, want lowercased", email)
	}
	if d := DaysTokenValid(tok); d != 1 {
		t.Errorf("DaysTokenValid = %d, want 1 (48h)", d)
	}
}

func TestResetTokenNotAConfirmToken(t *testing.T) {
	setTokenEnv(t, newTestKeys(t), "60m")
	tok, err := CreateResetToken(uuid.New(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ValidateEmailConfirmToken(tok); err == nil {
		t.Fatal("reset token accepted as email confirm token")
	}
}

func TestValidateEmailConfirmTokenRejects(t *testing.T) {
	keys := newTestKeys(t)
	other := newTestKeys(t)
	setTokenEnv(t, keys, "60m")
	now := time.Now()
	claims := func(mut func(jwt.MapClaims)) jwt.MapClaims {
		c := jwt.MapClaims{
			"purpose": confirmEmailPurpose,
			"uid":     uuid.New().String(),
			"email":   "a@b.com",
			"exp":     now.Add(time.Hour).Unix(),
		}
		if mut != nil {
			mut(c)
		}
		return c
	}
	rs := func(k *rsa.PrivateKey, c jwt.MapClaims) string {
		return signWith(t, jwt.SigningMethodRS256, k, c)
	}

	tests := []struct {
		name string
		tok  string
	}{
		{"expired", rs(keys.priv, claims(func(c jwt.MapClaims) { c["exp"] = now.Add(-time.Hour).Unix() }))},
		{"different key", rs(other.priv, claims(nil))},
		{"HS256", signWith(t, jwt.SigningMethodHS256, []byte("s"), claims(nil))},
		{"wrong purpose", rs(keys.priv, claims(func(c jwt.MapClaims) { c["purpose"] = "other" }))},
		{"missing purpose", rs(keys.priv, claims(func(c jwt.MapClaims) { delete(c, "purpose") }))},
		{"missing uid", rs(keys.priv, claims(func(c jwt.MapClaims) { delete(c, "uid") }))},
		{"missing email", rs(keys.priv, claims(func(c jwt.MapClaims) { delete(c, "email") }))},
		{"bad uid", rs(keys.priv, claims(func(c jwt.MapClaims) { c["uid"] = "xyz" }))},
		{"garbage", "garbage"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := ValidateEmailConfirmToken(tt.tok); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestDaysTokenValid(t *testing.T) {
	setTokenEnv(t, newTestKeys(t), "60m")
	mk := func(d time.Duration) string {
		tok, err := CreateToken(d, nil)
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	payload := func(s string) string {
		return "h." + base64.RawURLEncoding.EncodeToString([]byte(s)) + ".s"
	}
	tests := []struct {
		name string
		tok  string
		want int
	}{
		{"10 days", mk(10*24*time.Hour + time.Hour), 10},
		{"under a day", mk(5 * time.Hour), 0},
		{"two parts", "a.b", 0},
		{"bad base64", "a.!!!.c", 0},
		{"bad json", payload("{"), 0},
		{"no exp", payload(`{"sub":"x"}`), 0},
		{"exp wrong type", payload(`{"exp":"soon"}`), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DaysTokenValid(tt.tok); got != tt.want {
				t.Errorf("DaysTokenValid = %d, want %d", got, tt.want)
			}
		})
	}
	if got := DaysTokenValid(strings.Repeat("x", 5)); got != 0 {
		t.Errorf("garbage = %d", got)
	}
}

func TestResetTokenRoundTrip(t *testing.T) {
	setTokenEnv(t, newTestKeys(t), "60m")
	id := uuid.New()
	tok, err := CreateResetToken(id, 3)
	if err != nil {
		t.Fatal(err)
	}
	gotID, ver, err := ValidateResetToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if gotID != id || ver != 3 {
		t.Errorf("got (%s, %d), want (%s, 3)", gotID, ver, id)
	}
}

func TestValidateResetTokenRejects(t *testing.T) {
	keys := newTestKeys(t)
	setTokenEnv(t, keys, "60m")
	good := jwt.MapClaims{"purpose": resetPasswordPurpose, "uid": uuid.New().String(), "exp": time.Now().Add(time.Hour).Unix()}
	expired := jwt.MapClaims{"purpose": resetPasswordPurpose, "uid": uuid.New().String(), "exp": time.Now().Add(-time.Hour).Unix()}
	badUID := jwt.MapClaims{"purpose": resetPasswordPurpose, "uid": "nope", "exp": time.Now().Add(time.Hour).Unix()}
	if _, _, err := ValidateResetToken(signWith(t, jwt.SigningMethodRS256, keys.priv, good)); err != nil {
		t.Fatalf("control token rejected: %v", err)
	}
	for name, tok := range map[string]string{
		"expired":   signWith(t, jwt.SigningMethodRS256, keys.priv, expired),
		"bad uid":   signWith(t, jwt.SigningMethodRS256, keys.priv, badUID),
		"other key": signWith(t, jwt.SigningMethodRS256, newTestKeys(t).priv, good),
		"garbage":   "x.y.z",
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := ValidateResetToken(tok); err == nil {
				t.Error("expected error")
			}
		})
	}
}

// Keys may be stored as PKCS#8 as well as PKCS#1.
func TestResetTokenAcceptsPKCS8Key(t *testing.T) {
	keys := newTestKeys(t)
	der, err := x509.MarshalPKCS8PrivateKey(keys.priv)
	if err != nil {
		t.Fatal(err)
	}
	keys.privEnv = base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	setTokenEnv(t, keys, "")
	id := uuid.New()
	tok, err := CreateResetToken(id, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got, _, err := ValidateResetToken(tok); err != nil || got != id {
		t.Errorf("ValidateResetToken = %v, %v", got, err)
	}
}

func TestCreateResetTokenNeedsAPrivateKey(t *testing.T) {
	keys := newTestKeys(t)
	for name, priv := range map[string]string{"missing": "", "not base64": "%%%", "not PEM": base64.StdEncoding.EncodeToString([]byte("nope"))} {
		t.Run(name, func(t *testing.T) {
			setTokenEnv(t, keys, "")
			t.Setenv("ACCESS_TOKEN_PRIVATE_KEY", priv)
			if _, err := CreateResetToken(uuid.New(), 0); err == nil {
				t.Error("expected error")
			}
		})
	}
}

// A session cookie is a random string, never a valid reset token.
func TestSessionTokenIsNotAResetToken(t *testing.T) {
	setTokenEnv(t, newTestKeys(t), "")
	if _, _, err := ValidateResetToken("QkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCQkI"); err == nil {
		t.Error("ValidateResetToken accepted a random session-style token")
	}
}
