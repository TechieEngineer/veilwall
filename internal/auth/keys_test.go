package auth

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestIssue(t *testing.T) {
	k, raw, e := Issue("integration", []string{Read})
	if e != nil || Hash(raw) != k.KeyHash || !Allows(k, Read, time.Now()) || Allows(k, Simulate, time.Now()) {
		t.Fatal(e)
	}
	b, _ := json.Marshal(k)
	if strings.Contains(string(b), k.KeyHash) || strings.Contains(string(b), raw) {
		t.Fatal("secret serialized")
	}
	k.Revoked = true
	if Allows(k, Read, time.Now()) {
		t.Fatal("revoked key allowed")
	}
}
