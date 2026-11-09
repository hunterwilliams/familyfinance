package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	LineUserID  string    `json:"line_user_id"`
	HouseholdID uuid.UUID `json:"household_id"`
	MemberID    uuid.UUID `json:"member_id"`
	DisplayName string    `json:"display_name"`
	Dev         bool      `json:"dev,omitempty"`
	jwt.RegisteredClaims
}

func Issue(secret string, c Claims) (string, error) {
	c.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString([]byte(secret))
}

func Parse(secret, token string) (Claims, error) {
	var claims Claims
	parsed, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return Claims{}, err
	}
	if !parsed.Valid {
		return Claims{}, fmt.Errorf("invalid token")
	}
	return claims, nil
}
