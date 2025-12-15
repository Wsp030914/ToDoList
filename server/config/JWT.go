package config

import (
	"os"
	"time"
)

var (
	Secret    = pickSecret()
	Issuer    = getenv("JWT_ISSUER", "todo-api")
	Audience  = getenv("JWT_AUDIENCE", "todo-frontend")
	AccessTTL = mustParseDuration(getenv("JWT_ACCESS_TTL", "24h"))
)

func pickSecret() string {
	sec := os.Getenv("JWT_SECRET")
	if sec == "" {
		if os.Getenv("GO_ENV") == "production" {
			panic("JWT_SECRET is required in production")
		}
		sec = "dev-only-secret-please-change-me-32bytes-min!"
	}
	if len(sec) < 32 {
		panic("JWT_SECRET too short, need >= 32 bytes")
	}
	return sec
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}
