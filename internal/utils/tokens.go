package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"go-starter/internal/logger"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	accessTokenPrivateKey = os.Getenv("ACCESS_TOKEN_PRIVATE_KEY")
	accessTokenPublicKey  = os.Getenv("ACCESS_TOKEN_PUBLIC_KEY")
	accessTokenExpiresIn  = os.Getenv("ACCESS_TOKEN_EXPIRES_IN")
)

func CreateToken(userId uuid.UUID) (string, error) {
	decodedPrivateKey, err := base64.StdEncoding.DecodeString(accessTokenPrivateKey)

	if err != nil {
		return "", fmt.Errorf("could not decode key: %w", err)
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(decodedPrivateKey)

	if err != nil {
		return "", fmt.Errorf("create: parse key: %w", err)
	}
	expiresIn, err := time.ParseDuration(accessTokenExpiresIn)
	if err != nil {
		return "", fmt.Errorf("invalid expiry duration: %w", err)
	}

	now := time.Now().UTC()

	claims := make(jwt.MapClaims)
	claims["sub"] = userId
	claims["exp"] = now.Add(expiresIn).Unix()
	claims["iat"] = now.Unix()
	claims["nbf"] = now.Unix()

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(key)

	if err != nil {
		return "", fmt.Errorf("create: sign token: %w", err)
	}

	return token, nil
}

type TokenData struct {
	UserID    uuid.UUID
	AccountID uuid.UUID
}

func ValidateToken(token string) (*TokenData, error) {
	decodedPublicKey, err := base64.StdEncoding.DecodeString(accessTokenPublicKey)
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
	if claims["acc"] == nil {
		return nil, fmt.Errorf("validate: no acc")
	}

	userId, err := uuid.Parse(claims["sub"].(string))
	accountId, err := uuid.Parse(claims["acc"].(string))
	if err != nil {
		return nil, fmt.Errorf("validate: unable to parse tokens: %s", err)
	}

	tokenData := TokenData{
		UserID:    userId,
		AccountID: accountId,
	}
	return &tokenData, nil
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
