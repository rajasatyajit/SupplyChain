package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Key format: sc_{env}_{id}_{secret}
// - id: 12 url-safe chars
// - secret: 32 url-safe chars
func GenerateAPIKey(env string) (id string, rawKey string, secretHash []byte, err error) {
	id, secret, err := randomToken(12), randomToken(32), error(nil)
	if id == "" || secret == "" {
		return "", "", nil, fmt.Errorf("failed to generate token")
	}
	rawKey = fmt.Sprintf("sc_%s_%s_%s", env, id, secret)
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", "", nil, err
	}
	return id, rawKey, hash, nil
}

// ParseAPIKey splits into env, id, secret
func ParseAPIKey(raw string) (env string, id string, secret string, ok bool) {
	parts := strings.Split(raw, "_")
	if len(parts) != 4 || parts[0] != "sc" {
		return "", "", "", false
	}
	return parts[1], parts[2], parts[3], true
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	// URL-safe base64 without padding, then trim to n chars
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > n {
		return s[:n]
	}
	return s
}
