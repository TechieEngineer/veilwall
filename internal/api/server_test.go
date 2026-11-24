package api

import (
	"bytes"
	"encoding/json"
	"github.com/veilwall/veilwall/internal/firewall"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLocalMutationGuard(t *testing.T) {
	m := firewall.NewManager(firewall.NewMockEngine(), nil, time.Minute)
	defer m.Shutdown()
	h, e := New(m, nil, "demo", "127.0.0.1:8080", http.NotFoundHandler())
	if e != nil {
		t.Fatal(e)
	}
	call := func(method, path, host, origin, token string, body []byte) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://"+host+path, bytes.NewReader(body))
		r.RemoteAddr = "127.0.0.1:12345"
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Veilwall-Token", token)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	host := "127.0.0.1:8080"
	w := call("GET", "/api/v1/bootstrap", host, "", "", nil)
	var b map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &b)
	if w.Code != 200 || b["token"] == "" {
		t.Fatal(w.Code)
	}
	body := []byte(`{"rules":[],"revision":0}`)
	if w = call("POST", "/api/v1/rules/apply", host, "", "", body); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w = call("POST", "/api/v1/rules/apply", host, "https://evil.example", b["token"], body); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w = call("GET", "/api/v1/bootstrap", "evil.example", "", "", nil); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w = call("POST", "/api/v1/rules/apply", host, "http://"+host, b["token"], body); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
}
