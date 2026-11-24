package database

import (
	"context"
	"github.com/veilwall/veilwall/internal/models"
	"path/filepath"
	"testing"
)

func TestPersistence(t *testing.T) {
	ctx := context.Background()
	p := filepath.Join(t.TempDir(), "test.db")
	d, e := Open(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	r := models.Rule{ID: "r1", Name: "test", Direction: "input", Protocol: "tcp", Action: "drop", Mode: "shadow"}
	if e = d.Commit(ctx, []models.Rule{r}, "tx"); e != nil {
		t.Fatal(e)
	}
	d.Close()
	d, e = Open(ctx, p)
	if e != nil {
		t.Fatal(e)
	}
	defer d.Close()
	rules, e := d.Rules(ctx)
	if e != nil || len(rules) != 1 || rules[0] != r {
		t.Fatal(rules, e)
	}
	logs, e := d.Logs(ctx)
	if e != nil || len(logs) != 1 || logs[0].Action != "confirmed" {
		t.Fatal(logs, e)
	}
}
