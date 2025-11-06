package auth

import (
	"time"

	"NewStudent/config"
	"github.com/golang-jwt/jwt/v5"
)

var (
	// 放到配置里更安全
	secret    = []byte(config.Secret)
	issuer    = config.Issuer
	audience  = config.Audience
	accessTTL = config.AccessTTL // Access Token 有效期
)

type Claims struct {
	UID      int    `json:"uid"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// 生成 Access Token
func GenerateAccessToken(uid int, username string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(accessTTL)
	claims := &Claims{
		UID:      uid,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  []string{audience},
			Subject:   "access",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			ID:        "", // 可选 jti
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	return signed, exp, err
}

// 解析校验
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
