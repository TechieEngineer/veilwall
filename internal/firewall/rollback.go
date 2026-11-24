package firewall

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/veilwall/veilwall/internal/models"
	"sync"
	"time"
)

var ErrConflict = errors.New("transaction conflict or expired confirmation")

// Store atomically commits the confirmed rules and audit event. Pending state never persists.
type Store interface {
	Commit(context.Context, []models.Rule, string) error
	Audit(context.Context, string, string, string) error
}
type Transaction struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
}
type State struct {
	Rules     []models.Rule `json:"rules"`
	Pending   *Transaction  `json:"pending"`
	LastEvent string        `json:"last_event"`
	Revision  uint64        `json:"revision"`
}
type pending struct {
	info     Transaction
	previous []models.Rule
	timer    *time.Timer
}
type Manager struct {
	mu       sync.Mutex
	engine   Engine
	store    Store
	timeout  time.Duration
	p        *pending
	closed   bool
	last     string
	revision uint64
}

func NewManager(e Engine, s Store, timeout time.Duration) *Manager {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Manager{engine: e, store: s, timeout: timeout, last: "ready"}
}
func (m *Manager) State(ctx context.Context) (State, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, e := m.engine.Snapshot(ctx)
	s := State{Rules: r, LastEvent: m.last, Revision: m.revision}
	if m.p != nil {
		v := m.p.info
		s.Pending = &v
	}
	return s, e
}
func (m *Manager) Apply(ctx context.Context, r []models.Rule, revision uint64) (Transaction, error) {
	if e := models.ValidateRules(r); e != nil {
		return Transaction{}, e
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.p != nil || m.revision != revision {
		return Transaction{}, ErrConflict
	}
	old, e := m.engine.Snapshot(ctx)
	if e != nil {
		return Transaction{}, e
	}
	b := make([]byte, 16)
	if _, e = rand.Read(b); e != nil {
		return Transaction{}, e
	}
	id := hex.EncodeToString(b)
	if e = m.engine.Apply(ctx, r); e != nil {
		return Transaction{}, e
	}
	p := &pending{info: Transaction{id, time.Now().Add(m.timeout)}, previous: old}
	m.p = p
	m.revision++
	m.last = "staged"
	p.timer = time.AfterFunc(m.timeout, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.p == p {
			_ = m.restoreLocked("expired")
		}
	})
	m.audit("staged", id)
	return p.info, nil
}
func (m *Manager) Confirm(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.p == nil || m.p.info.ID != id {
		return ErrConflict
	}
	if !time.Now().Before(m.p.info.ExpiresAt) {
		_ = m.restoreLocked("expired")
		return ErrConflict
	}
	r, e := m.engine.Snapshot(ctx)
	if e != nil {
		return e
	}
	if m.store != nil {
		if e = m.store.Commit(ctx, r, id); e != nil {
			return e
		}
	}
	m.p.timer.Stop()
	m.p = nil
	m.revision++
	m.last = "confirmed"
	return nil
}
func (m *Manager) Rollback(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.p == nil || m.p.info.ID != id {
		return ErrConflict
	}
	return m.restoreLocked("rolled_back")
}
func (m *Manager) restoreLocked(event string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if e := m.engine.Restore(ctx, m.p.previous); e != nil {
		m.last = "restore_failed"
		// Retry a failed restore while the daemon is still running.
		if !m.closed {
			p := m.p
			p.timer.Stop()
			p.timer = time.AfterFunc(time.Second, func() {
				m.mu.Lock()
				defer m.mu.Unlock()
				if m.p == p {
					_ = m.restoreLocked(event)
				}
			})
		}
		return e
	}
	id := m.p.info.ID
	m.p.timer.Stop()
	m.p = nil
	m.revision++
	m.last = event
	m.audit(event, id)
	return nil
}
func (m *Manager) audit(action, id string) {
	if m.store != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = m.store.Audit(ctx, action, id, "local demo")
	}
}
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	if m.p != nil {
		return m.restoreLocked("shutdown_rollback")
	}
	return nil
}
