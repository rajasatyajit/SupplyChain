package auth

import (
	"strings"
	"testing"
)

func TestGenerateAndParseAPIKey(t *testing.T) {
	id, raw, hash, err := GenerateAPIKey("test")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if id == "" || raw == "" {
		t.Fatalf("expected non-empty id and raw")
	}
	if !strings.HasPrefix(raw, "sc.test.") {
		t.Fatalf("unexpected prefix: %s", raw)
	}
	env, parsedID, secret, ok := ParseAPIKey(raw)
	if !ok {
		t.Fatalf("parse failed")
	}
	if env != "test" || parsedID != id || secret == "" {
		t.Fatalf("bad parse: env=%s id=%s secret=%s", env, parsedID, secret)
	}
	if len(hash) == 0 {
		t.Fatalf("expected hash")
	}
}
