package utils

import (
	"ToDoList/server/config"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	secret    = []byte(config.Secret)
	issuer    = config.Issuer
	audience  = config.Audience
	accessTTL = config.AccessTTL // Access Token 有效期
)

type Claims struct {
	UID      int    `json:"uid"`
	Username string `json:"username"`
	Ver      int    `json:"ver"` //
	jwt.RegisteredClaims
}

func GenerateAccessToken(uid int, username string, tokenVersion int) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(accessTTL)
	jti := fmt.Sprintf("acc_%d_%d", uid, now.UnixNano())
	claims := &Claims{
		UID:      uid,
		Username: username,
		Ver:      tokenVersion, // ← 对应上面的 Claims.Ver
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  []string{audience},
			Subject:   "access",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(secret)
	return signed, exp, err
}

func Parse(tokenStr string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secret, nil
	}, jwt.WithAudience(audience), jwt.WithIssuer(issuer))
	if err != nil {
		return nil, err
	}
	if c, ok := tok.Claims.(*Claims); ok && tok.Valid {
		return c, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
