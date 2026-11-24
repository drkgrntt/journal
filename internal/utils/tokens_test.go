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

func setTokenEnv(t *testing.T, k testKeys, expiresIn string) {
	t.Helper()
	t.Setenv("ACCESS_TOKEN_PRIVATE_KEY", k.privEnv)
	t.Setenv("ACCESS_TOKEN_PUBLIC_KEY", k.pubEnv)
	t.Setenv("ACCESS_TOKEN_EXPIRES_IN", expiresIn)
}

func signWith(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestAccessTokenRoundTrip(t *testing.T) {
	setTokenEnv(t, newTestKeys(t), "60m")
	id := uuid.New()
	tok, err := CreateAccessToken(id, 0)
	if err != nil {
		t.Fatal(err)
	}
	data, err := ValidateToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if data.UserID != id {
		t.Errorf("UserID = %s, want %s", data.UserID, id)
	}
}

func TestCreateAccessTokenErrors(t *testing.T) {
	keys := newTestKeys(t)
	tests := []struct {
		name string
		priv string
		exp  string
	}{
		{"invalid expiry", keys.privEnv, "soon"},
		{"missing expiry", keys.privEnv, ""},
		{"missing private key", "", "60m"},
		{"private key not base64", "%%%", "60m"},
		{"private key not PEM", base64.StdEncoding.EncodeToString([]byte("nope")), "60m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setTokenEnv(t, keys, tt.exp)
			t.Setenv("ACCESS_TOKEN_PRIVATE_KEY", tt.priv)
			if _, err := CreateAccessToken(uuid.New(), 0); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestCreateAccessTokenAcceptsPKCS8(t *testing.T) {
	keys := newTestKeys(t)
	der, err := x509.MarshalPKCS8PrivateKey(keys.priv)
	if err != nil {
		t.Fatal(err)
	}
	pk8 := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	keys.privEnv = base64.StdEncoding.EncodeToString(pk8)
	setTokenEnv(t, keys, "1h")
	id := uuid.New()
	tok, err := CreateAccessToken(id, 0)
	if err != nil {
		t.Fatal(err)
	}
	if data, err := ValidateToken(tok); err != nil || data.UserID != id {
		t.Errorf("ValidateToken = %v, %v", data, err)
	}
}

func TestValidateTokenRejects(t *testing.T) {
	keys := newTestKeys(t)
	other := newTestKeys(t)
	setTokenEnv(t, keys, "60m")
	id := uuid.New().String()
	now := time.Now()
	valid := func() jwt.MapClaims {
		return jwt.MapClaims{"sub": id, "exp": now.Add(time.Hour).Unix(), "iat": now.Unix()}
	}
	expired := valid()
	expired["exp"] = now.Add(-time.Hour).Unix()
	noSub := valid()
	delete(noSub, "sub")
	badSub := valid()
	badSub["sub"] = "not-a-uuid"

	tests := []struct {
		name string
		tok  string
	}{
		{"expired", signWith(t, jwt.SigningMethodRS256, keys.priv, expired)},
		{"different key", signWith(t, jwt.SigningMethodRS256, other.priv, valid())},
		{"HS256", signWith(t, jwt.SigningMethodHS256, []byte("secret"), valid())},
		{"none alg", signWith(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, valid())},
		{"no sub", signWith(t, jwt.SigningMethodRS256, keys.priv, noSub)},
		{"bad sub uuid", signWith(t, jwt.SigningMethodRS256, keys.priv, badSub)},
		{"garbage", "not.a.token"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if data, err := ValidateToken(tt.tok); err == nil {
				t.Errorf("accepted token, got %+v", data)
			}
		})
	}
}

func TestValidateTokenBadEnv(t *testing.T) {
	keys := newTestKeys(t)
	setTokenEnv(t, keys, "60m")
	tok, err := CreateAccessToken(uuid.New(), 0)
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
			if _, err := ValidateToken(tok); err == nil {
				t.Error("ValidateToken: expected error")
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

func TestEmailConfirmTokenNotALoginToken(t *testing.T) {
	setTokenEnv(t, newTestKeys(t), "60m")
	tok, err := CreateEmailConfirmToken(uuid.New(), "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	if data, err := ValidateToken(tok); err == nil {
		t.Fatalf("email confirm token accepted as login token: %+v", data)
	}
}

func TestAccessTokenNotAConfirmToken(t *testing.T) {
	setTokenEnv(t, newTestKeys(t), "60m")
	tok, err := CreateAccessToken(uuid.New(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ValidateEmailConfirmToken(tok); err == nil {
		t.Fatal("access token accepted as email confirm token")
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

func TestAccessTokenCarriesVersion(t *testing.T) {
	setTokenEnv(t, newTestKeys(t), "60m")
	tok, err := CreateAccessToken(uuid.New(), 7)
	if err != nil {
		t.Fatal(err)
	}
	data, err := ValidateToken(tok)
	if err != nil {
		t.Fatal(err)
	}
	if data.Version != 7 {
		t.Errorf("Version = %d, want 7", data.Version)
	}
}

func TestTokenWithoutVersionCountsAsZero(t *testing.T) {
	keys := newTestKeys(t)
	setTokenEnv(t, keys, "60m")
	id := uuid.New()
	old := signWith(t, jwt.SigningMethodRS256, keys.priv, jwt.MapClaims{"sub": id.String(), "exp": time.Now().Add(time.Hour).Unix()})
	data, err := ValidateToken(old)
	if err != nil {
		t.Fatal(err)
	}
	if data.Version != 0 {
		t.Errorf("Version = %d, want 0 for a token issued before versions", data.Version)
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

// The reset link is emailed, so it must never double as a login cookie.
func TestResetTokenIsNotASessionToken(t *testing.T) {
	setTokenEnv(t, newTestKeys(t), "60m")
	tok, err := CreateResetToken(uuid.New(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if data, err := ValidateToken(tok); err == nil {
		t.Errorf("ValidateToken accepted a reset token: %+v", data)
	}
}

// A stolen session cookie must not be usable to reset the password.
func TestSessionTokenIsNotAResetToken(t *testing.T) {
	setTokenEnv(t, newTestKeys(t), "60m")
	tok, err := CreateAccessToken(uuid.New(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ValidateResetToken(tok); err == nil {
		t.Error("ValidateResetToken accepted a session token")
	}
	conf, err := CreateEmailConfirmToken(uuid.New(), "a@b.co")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ValidateResetToken(conf); err == nil {
		t.Error("ValidateResetToken accepted a confirmation token")
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

// Even a token that somehow carries a "sub" must not log in if it is marked
// for another purpose.
func TestValidateTokenRejectsOtherPurposeEvenWithSub(t *testing.T) {
	keys := newTestKeys(t)
	setTokenEnv(t, keys, "60m")
	tok := signWith(t, jwt.SigningMethodRS256, keys.priv, jwt.MapClaims{
		"sub": uuid.New().String(), "purpose": resetPasswordPurpose, "exp": time.Now().Add(time.Hour).Unix(),
	})
	if data, err := ValidateToken(tok); err == nil {
		t.Errorf("accepted a purpose-marked token: %+v", data)
	}
}
