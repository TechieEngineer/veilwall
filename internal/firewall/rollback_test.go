package firewall

import (
	"context"
	"errors"
	"github.com/veilwall/veilwall/internal/models"
	"testing"
	"time"
)

func rule(id string) models.Rule {
	return models.Rule{ID: id, Name: id, Direction: "input", Protocol: "tcp", Port: 443, Mode: "shadow", Action: "drop"}
}
func TestRollbackAndConfirm(t *testing.T) {
	ctx := context.Background()
	e := NewMockEngine()
	_ = e.Apply(ctx, []models.Rule{rule("old")})
	m := NewManager(e, nil, 80*time.Millisecond)
	defer m.Shutdown()
	tx, err := m.Apply(ctx, []models.Rule{rule("new")}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = m.Apply(ctx, nil, 1); !errors.Is(err, ErrConflict) {
		t.Fatal(err)
	}
	end := time.Now().Add(2 * time.Second)
	for {
		state, _ := m.State(ctx)
		if state.Pending == nil {
			if state.Rules[0].ID != "old" || state.LastEvent != "expired" {
				t.Fatal(state)
			}
			break
		}
		if time.Now().After(end) {
			t.Fatal("timer failed")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !errors.Is(m.Confirm(ctx, tx.ID), ErrConflict) {
		t.Fatal("expired accepted")
	}
	s, _ := m.State(ctx)
	tx, err = m.Apply(ctx, []models.Rule{rule("keep")}, s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Confirm(ctx, tx.ID); err != nil {
		t.Fatal(err)
	}
	if err = m.Shutdown(); err != nil {
		t.Fatal(err)
	}
	s, _ = m.State(ctx)
	if s.Rules[0].ID != "keep" {
		t.Fatal(s)
	}
}
func TestShutdown(t *testing.T) {
	ctx := context.Background()
	e := NewMockEngine()
	m := NewManager(e, nil, time.Minute)
	_, err := m.Apply(ctx, []models.Rule{rule("new")}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Shutdown(); err != nil {
		t.Fatal(err)
	}
	s, _ := m.State(ctx)
	if len(s.Rules) != 0 || s.Pending != nil {
		t.Fatal(s)
	}
}
func TestInvalidAndStale(t *testing.T) {
	m := NewManager(NewMockEngine(), nil, time.Minute)
	defer m.Shutdown()
	ctx := context.Background()
	if _, e := m.Apply(ctx, []models.Rule{{ID: "bad"}}, 0); e == nil {
		t.Fatal("invalid accepted")
	}
	if _, e := m.Apply(ctx, nil, 42); !errors.Is(e, ErrConflict) {
		t.Fatal(e)
	}
}
