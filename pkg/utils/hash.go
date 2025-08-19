package utils

import (
	"crypto/sha1"
	"encoding/hex"
)

// HashString generates a SHA1 hash of a string
func HashString(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
