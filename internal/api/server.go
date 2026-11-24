package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/veilwall/veilwall/internal/auth"
	"github.com/veilwall/veilwall/internal/firewall"
	"github.com/veilwall/veilwall/internal/models"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"
)

type LogReader interface {
	Logs(context.Context) ([]models.AuditLog, error)
}
type Server struct {
	manager             *firewall.Manager
	logs                LogReader
	engine, token, host string
	started             time.Time
	streams             chan struct{}
	keys                auth.Store
}

func New(m *firewall.Manager, l LogReader, engine, host string, assets http.Handler) (http.Handler, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return nil, e
	}
	s := &Server{manager: m, logs: l, engine: engine, token: hex.EncodeToString(b), host: host, started: time.Now(), streams: make(chan struct{}, 8)}
	s.keys, _ = l.(auth.Store)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		jsonOut(w, 200, map[string]any{"status": "healthy", "version": "1.0.0-beta.1", "engine": engine, "enforcement": false})
	})
	mux.HandleFunc("GET /api/v1/bootstrap", func(w http.ResponseWriter, r *http.Request) { jsonOut(w, 200, map[string]string{"token": s.token}) })
	mux.HandleFunc("GET /api/v1/rules", s.state)
	mux.HandleFunc("GET /api/v1/stats", s.stats)
	mux.HandleFunc("GET /api/v1/audit", s.audit)
	mux.HandleFunc("GET /api/v1/telemetry", s.sample)
	mux.HandleFunc("GET /api/v1/telemetry/stream", s.stream)
	mux.HandleFunc("POST /api/v1/simulate", s.simulate)
	mux.HandleFunc("GET /api/v1/keys", s.listKeys)
	mux.HandleFunc("POST /api/v1/keys", s.issueKey)
	mux.HandleFunc("POST /api/v1/keys/{id}/revoke", s.revokeKey)
	mux.HandleFunc("GET /api/v1/integration/stats", s.integrationStats)
	mux.HandleFunc("GET /api/v1/integration/telemetry", s.integrationSample)
	mux.HandleFunc("POST /api/v1/integration/simulate", s.integrationSimulate)
	mux.HandleFunc("POST /api/v1/rules/apply", s.apply)
	mux.HandleFunc("POST /api/v1/transactions/{id}/confirm", s.confirm)
	mux.HandleFunc("POST /api/v1/transactions/{id}/rollback", s.rollback)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { problem(w, 404, "endpoint not found") })
	mux.Handle("/", assets)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		peer, _, e := net.SplitHostPort(r.RemoteAddr)
		ip, ie := netip.ParseAddr(peer)
		if e != nil || ie != nil || !ip.IsLoopback() || !strings.EqualFold(r.Host, s.host) {
			problem(w, 403, "local access only; use the printed address")
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+s.host {
			problem(w, 403, "origin denied")
			return
		}
		if site := r.Header.Get("Sec-Fetch-Site"); site == "cross-site" || site == "same-site" {
			problem(w, 403, "cross-origin request denied")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/integration/") {
			// Integration endpoints use a scoped bearer, never the browser session token.
			mux.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/api/v1/keys" && r.Method == "GET" && subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Veilwall-Token")), []byte(s.token)) != 1 {
			problem(w, 403, "local session token required")
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Veilwall-Token")), []byte(s.token)) != 1 {
				problem(w, 403, "invalid local session token")
				return
			}
			if r.Header.Get("Content-Type") != "application/json" {
				problem(w, 415, "application/json required")
				return
			}
		}
		mux.ServeHTTP(w, r)
	}), nil
}
func jsonOut(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func problem(w http.ResponseWriter, status int, msg string) {
	jsonOut(w, status, map[string]string{"error": msg})
}
func (s *Server) state(w http.ResponseWriter, r *http.Request) {
	v, e := s.manager.State(r.Context())
	if e != nil {
		problem(w, 500, "state unavailable")
		return
	}
	jsonOut(w, 200, v)
}
func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	v, e := s.manager.State(r.Context())
	if e != nil {
		problem(w, 500, "state unavailable")
		return
	}
	active, shadow := 0, 0
	for _, rule := range v.Rules {
		if rule.Mode == "active" {
			active++
		}
		if rule.Mode == "shadow" {
			shadow++
		}
	}
	jsonOut(w, 200, map[string]any{"engine": s.engine, "enforcement": false, "rules": len(v.Rules), "active": active, "shadow": shadow, "uptime_seconds": int64(time.Since(s.started).Seconds())})
}
func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	if s.logs == nil {
		jsonOut(w, 200, []models.AuditLog{})
		return
	}
	v, e := s.logs.Logs(r.Context())
	if e != nil {
		problem(w, 500, "audit unavailable")
		return
	}
	jsonOut(w, 200, v)
}
func (s *Server) apply(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	var body struct {
		Rules    []models.Rule `json:"rules"`
		Revision *uint64       `json:"revision"`
	}
	if e := d.Decode(&body); e != nil || body.Revision == nil || body.Rules == nil {
		problem(w, 400, "invalid payload: rules and revision required")
		return
	}
	if d.Decode(new(any)) != io.EOF {
		problem(w, 400, "one JSON object required")
		return
	}
	if e := models.ValidateRules(body.Rules); e != nil {
		problem(w, 400, e.Error())
		return
	}
	v, e := s.manager.Apply(r.Context(), body.Rules, *body.Revision)
	if e != nil {
		s.mutationError(w, e)
		return
	}
	jsonOut(w, 201, v)
}
func (s *Server) confirm(w http.ResponseWriter, r *http.Request) {
	if e := s.manager.Confirm(r.Context(), r.PathValue("id")); e != nil {
		s.mutationError(w, e)
		return
	}
	jsonOut(w, 200, map[string]string{"status": "confirmed"})
}
func (s *Server) rollback(w http.ResponseWriter, r *http.Request) {
	if e := s.manager.Rollback(r.PathValue("id")); e != nil {
		s.mutationError(w, e)
		return
	}
	jsonOut(w, 200, map[string]string{"status": "rolled_back"})
}
func (s *Server) mutationError(w http.ResponseWriter, e error) {
	if errors.Is(e, firewall.ErrConflict) {
		problem(w, 409, e.Error())
		return
	}
	problem(w, 500, "transaction failed")
}
