package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"journal/internal/logger"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// The token settings are read per call, not at package load, so tests can set
// them with t.Setenv.
func accessTokenPrivateKey() string { return os.Getenv("ACCESS_TOKEN_PRIVATE_KEY") }
func accessTokenPublicKey() string  { return os.Getenv("ACCESS_TOKEN_PUBLIC_KEY") }
func accessTokenExpiresIn() string  { return os.Getenv("ACCESS_TOKEN_EXPIRES_IN") }

func CreateToken(expiresIn time.Duration, additionalClaims map[string]any) (string, error) {
	decodedPrivateKey, err := base64.StdEncoding.DecodeString(accessTokenPrivateKey())

	if err != nil {
		return "", fmt.Errorf("could not decode key: %w", err)
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(decodedPrivateKey)

	if err != nil {
		return "", fmt.Errorf("create: parse key: %w", err)
	}

	now := time.Now().UTC()

	claims := make(jwt.MapClaims)
	claims["exp"] = now.Add(expiresIn).Unix()
	claims["iat"] = now.Unix()
	claims["nbf"] = now.Unix()
	for k, v := range additionalClaims {
		claims[k] = v
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)

	if err != nil {
		return "", fmt.Errorf("create: sign token: %w", err)
	}

	return token, nil
}

func CreateAccessToken(userId uuid.UUID) (string, error) {
	expiresIn, err := time.ParseDuration(accessTokenExpiresIn())
	if err != nil {
		return "", fmt.Errorf("invalid expiry duration: %w", err)
	}

	claims := make(map[string]any)
	claims["sub"] = userId

	token, err := CreateToken(expiresIn, claims)

	if err != nil {
		return "", fmt.Errorf("create: sign token: %w", err)
	}

	return token, nil
}

type TokenData struct {
	UserID uuid.UUID
}

func ValidateToken(token string) (*TokenData, error) {
	decodedPublicKey, err := base64.StdEncoding.DecodeString(accessTokenPublicKey())
	if err != nil {
		return nil, fmt.Errorf("could not decode: %w", err)
	}

	key, err := jwt.ParseRSAPublicKeyFromPEM(decodedPublicKey)

	if err != nil {
		return nil, fmt.Errorf("validate: parse key: %w", err)
	}

	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected method: %s", t.Header["alg"])
		}
		return key, nil
	})

	if err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, fmt.Errorf("validate: invalid token")
	}

	if claims["sub"] == nil {
		return nil, fmt.Errorf("validate: no sub")
	}

	userId, err := uuid.Parse(claims["sub"].(string))
	if err != nil {
		return nil, fmt.Errorf("validate: unable to parse tokens: %s", err)
	}

	tokenData := TokenData{
		UserID: userId,
	}
	return &tokenData, nil
}

const confirmEmailPurpose = "confirm-email"

// CreateEmailConfirmToken makes a link token for confirming one address. It
// deliberately has no "sub" claim, so ValidateToken rejects it and it can never
// be used as a login cookie. It carries the address it was issued for, so a
// later change of address invalidates it.
func CreateEmailConfirmToken(userId uuid.UUID, email string) (string, error) {
	return CreateToken(time.Hour*48, map[string]any{
		"purpose": confirmEmailPurpose,
		"uid":     userId.String(),
		"email":   strings.ToLower(email),
	})
}

// ValidateEmailConfirmToken returns the user ID and (lowercased) address a
// confirmation token was issued for.
func ValidateEmailConfirmToken(token string) (uuid.UUID, string, error) {
	decodedPublicKey, err := base64.StdEncoding.DecodeString(accessTokenPublicKey())
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("could not decode: %w", err)
	}
	key, err := jwt.ParseRSAPublicKeyFromPEM(decodedPublicKey)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("validate: parse key: %w", err)
	}

	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected method: %s", t.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("validate: %w", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return uuid.Nil, "", fmt.Errorf("validate: invalid token")
	}
	purpose, _ := claims["purpose"].(string)
	uid, _ := claims["uid"].(string)
	email, _ := claims["email"].(string)
	if purpose != confirmEmailPurpose || uid == "" || email == "" {
		return uuid.Nil, "", fmt.Errorf("validate: not a confirmation token")
	}
	userId, err := uuid.Parse(uid)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("validate: bad user id")
	}
	return userId, email, nil
}

func DaysTokenValid(token string) int {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		logger.Error("Invalid token format")
		return 0
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		logger.Error("Error decoding token payload: ", err)
		return 0
	}

	var claims map[string]interface{}
	err = json.Unmarshal(payload, &claims)
	if err != nil {
		logger.Error("Error unmarshalling token payload: ", err)
		return 0
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		logger.Error("Error: exp claim is not a float64")
		return 0
	}

	expTime := time.Unix(int64(exp), 0)
	return int(time.Until(expTime).Hours() / 24)
}
