package api

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/veilwall/veilwall/internal/auth"
	"github.com/veilwall/veilwall/internal/firewall"
	"github.com/veilwall/veilwall/internal/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type memKeys struct{ key models.APIKey }

func (m *memKeys) Logs(context.Context) ([]models.AuditLog, error)    { return nil, nil }
func (m *memKeys) CreateKey(_ context.Context, k models.APIKey) error { m.key = k; return nil }
func (m *memKeys) Keys(context.Context) ([]models.APIKey, error)      { return []models.APIKey{m.key}, nil }
func (m *memKeys) KeyByHash(_ context.Context, h string) (models.APIKey, error) {
	if h != m.key.KeyHash {
		return models.APIKey{}, errors.New("not found")
	}
	return m.key, nil
}
func (m *memKeys) RevokeKey(_ context.Context, id string) error {
	if id != m.key.ID {
		return errors.New("not found")
	}
	m.key.Revoked = true
	return nil
}
func TestScopedIntegration(t *testing.T) {
	store := &memKeys{}
	manager := firewall.NewManager(firewall.NewMockEngine(), nil, time.Minute)
	defer manager.Shutdown()
	h, e := New(manager, store, "demo", "127.0.0.1:8080", http.NotFoundHandler())
	if e != nil {
		t.Fatal(e)
	}
	call := func(method, path, body, local, bearer string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, strings.NewReader(body))
		r.RemoteAddr = "127.0.0.1:1234"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Veilwall-Token", local)
		if bearer != "" {
			r.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	var boot map[string]string
	json.Unmarshal(call("GET", "/api/v1/bootstrap", "", "", "").Body.Bytes(), &boot)
	w := call("POST", "/api/v1/keys", `{"name":"demo","scopes":["read"]}`, boot["token"], "")
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var issued struct {
		Token string `json:"token"`
	}
	json.Unmarshal(w.Body.Bytes(), &issued)
	if store.key.KeyHash != auth.Hash(issued.Token) {
		t.Fatal("bad hash")
	}
	for _, v := range []struct {
		method, path, body, token string
		status                    int
	}{{"GET", "/api/v1/integration/stats", "", "", 401}, {"GET", "/api/v1/integration/stats", "", issued.Token, 200}, {"GET", "/api/v1/integration/telemetry", "", issued.Token, 200}, {"POST", "/api/v1/integration/simulate", "{}", issued.Token, 403}, {"POST", "/api/v1/rules/apply", "{}", issued.Token, 403}, {"GET", "/api/v1/keys", "", issued.Token, 403}} {
		if w := call(v.method, v.path, v.body, "", v.token); w.Code != v.status {
			t.Fatalf("%s: %d %s", v.path, w.Code, w.Body.String())
		}
	}
	if w := call("POST", "/api/v1/keys/"+store.key.ID+"/revoke", "{}", boot["token"], ""); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := call("GET", "/api/v1/integration/stats", "", "", issued.Token); w.Code != 403 {
		t.Fatal(w.Code)
	}
}
