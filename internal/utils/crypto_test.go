package utils

import (
	"encoding/base64"
	"strings"
	"testing"
)

const testCryptoKey = "0123456789abcdef0123456789abcdef" // 32 bytes

func setCryptoKey(t *testing.T, key string) {
	t.Helper()
	t.Setenv("CRYPTO_KEY", key)
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	setCryptoKey(t, testCryptoKey)
	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"ascii", "hello world"},
		{"unicode", "héllo 世界 🌧️ ok"},
		{"multiline", "line one\nline two\n"},
		{"long", strings.Repeat("a", 10000)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := Encrypt(tt.in)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}
			got, err := Decrypt(enc)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if got != tt.in {
				t.Errorf("round trip = %q, want %q", got, tt.in)
			}
		})
	}
}

func TestEncryptKeyLengths(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"16 bytes", strings.Repeat("k", 16), false},
		{"24 bytes", strings.Repeat("k", 24), false},
		{"32 bytes", strings.Repeat("k", 32), false},
		{"empty", "", true},
		{"15 bytes", strings.Repeat("k", 15), true},
		{"33 bytes", strings.Repeat("k", 33), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setCryptoKey(t, tt.key)
			enc, err := Encrypt("x")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Encrypt err = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if _, err := Decrypt("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"); err == nil {
					t.Error("Decrypt with bad key should error")
				}
			} else if got, err := Decrypt(enc); err != nil || got != "x" {
				t.Errorf("Decrypt = %q, %v", got, err)
			}
		})
	}
}

func TestEncryptIsRandomized(t *testing.T) {
	setCryptoKey(t, testCryptoKey)
	a, err := Encrypt("same")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Encrypt("same")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Error("two encryptions of the same plaintext were identical")
	}
}

func TestDecryptFailures(t *testing.T) {
	setCryptoKey(t, testCryptoKey)
	good, err := Encrypt("secret")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := base64.StdEncoding.DecodeString(good)
	flip := func(i int) string {
		b := append([]byte(nil), raw...)
		b[i] ^= 0xff
		return base64.StdEncoding.EncodeToString(b)
	}

	tests := []struct {
		name string
		in   string
	}{
		{"invalid base64", "!!!not base64!!!"},
		{"empty", ""},
		{"too short", base64.StdEncoding.EncodeToString([]byte("short"))},
		{"tampered nonce", flip(0)},
		{"tampered body", flip(len(raw) - 1)},
		{"truncated", base64.StdEncoding.EncodeToString(raw[:len(raw)-1])},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := Decrypt(tt.in); err == nil {
				t.Errorf("Decrypt succeeded with %q", got)
			}
		})
	}
}

func TestDecryptWrongKey(t *testing.T) {
	setCryptoKey(t, testCryptoKey)
	enc, err := Encrypt("secret")
	if err != nil {
		t.Fatal(err)
	}
	setCryptoKey(t, "ffffffffffffffffffffffffffffffff")
	if got, err := Decrypt(enc); err == nil {
		t.Errorf("Decrypt with wrong key succeeded: %q", got)
	}
}
