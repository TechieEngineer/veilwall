package database

import (
	"context"
	"github.com/veilwall/veilwall/internal/auth"
	"testing"
	"time"
)

func TestPersistentKeys(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/keys.db"
	db, e := Open(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	k, raw, e := auth.Issue("test", []string{auth.Read})
	if e != nil {
		t.Fatal(e)
	}
	if e = db.CreateKey(ctx, k); e != nil {
		t.Fatal(e)
	}
	db.Close()
	db, e = Open(ctx, path)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	got, e := db.KeyByHash(ctx, auth.Hash(raw))
	if e != nil || !auth.Allows(got, auth.Read, time.Now()) {
		t.Fatal(e)
	}
	if e = db.RevokeKey(ctx, k.ID); e != nil {
		t.Fatal(e)
	}
	got, e = db.KeyByHash(ctx, auth.Hash(raw))
	if e != nil || !got.Revoked {
		t.Fatal(e)
	}
}
