package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"github.com/veilwall/veilwall/internal/models"
	"strings"
	"time"
)

const Read = "read"
const Simulate = "simulate"

type Store interface {
	CreateKey(context.Context, models.APIKey) error
	Keys(context.Context) ([]models.APIKey, error)
	KeyByHash(context.Context, string) (models.APIKey, error)
	RevokeKey(context.Context, string) error
}

func Hash(raw string) string { sum := sha256.Sum256([]byte(raw)); return hex.EncodeToString(sum[:]) }
func Issue(name string, scopes []string) (models.APIKey, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 || len(scopes) == 0 || len(scopes) > 2 {
		return models.APIKey{}, "", errors.New("name and one or two scopes required")
	}
	seen := map[string]bool{}
	for _, s := range scopes {
		if (s != Read && s != Simulate) || seen[s] {
			return models.APIKey{}, "", errors.New("invalid or duplicate scope")
		}
		seen[s] = true
	}
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return models.APIKey{}, "", e
	}
	raw := "vw_" + hex.EncodeToString(b)
	id := make([]byte, 16)
	if _, e := rand.Read(id); e != nil {
		return models.APIKey{}, "", e
	}
	now := time.Now().UTC()
	expires := now.Add(30 * 24 * time.Hour)
	return models.APIKey{ID: hex.EncodeToString(id), Name: name, Prefix: raw[:11], KeyHash: Hash(raw), Scopes: scopes, CreatedAt: now, ExpiresAt: &expires}, raw, nil
}
func Allows(k models.APIKey, scope string, now time.Time) bool {
	if k.Revoked || k.ExpiresAt == nil || !now.Before(*k.ExpiresAt) {
		return false
	}
	for _, s := range k.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}
