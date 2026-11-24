package firewall

import (
	"context"
	"github.com/veilwall/veilwall/internal/models"
	"slices"
	"sync"
)

type Engine interface {
	Name() string
	Snapshot(context.Context) ([]models.Rule, error)
	Apply(context.Context, []models.Rule) error
	Restore(context.Context, []models.Rule) error
}
type MockEngine struct {
	mu    sync.RWMutex
	rules []models.Rule
}

func NewMockEngine() *MockEngine   { return &MockEngine{rules: []models.Rule{}} }
func (m *MockEngine) Name() string { return "demo" }
func (m *MockEngine) Snapshot(ctx context.Context) ([]models.Rule, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	return slices.Clone(m.rules), nil
}
func (m *MockEngine) Apply(ctx context.Context, r []models.Rule) error {
	if e := models.ValidateRules(r); e != nil {
		return e
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return e
	}
	m.rules = append([]models.Rule{}, r...)
	return nil
}
func (m *MockEngine) Restore(ctx context.Context, r []models.Rule) error { return m.Apply(ctx, r) }
