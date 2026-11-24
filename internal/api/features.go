package api

import (
	"encoding/json"
	"github.com/veilwall/veilwall/internal/auth"
	"github.com/veilwall/veilwall/internal/simulation"
	"github.com/veilwall/veilwall/internal/telemetry"
	"io"
	"net/http"
	"strings"
	"time"
)

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(new(any)) != io.EOF {
		problem(w, 400, "invalid JSON payload")
		return false
	}
	return true
}
func (s *Server) sample(w http.ResponseWriter, r *http.Request) {
	st, e := s.manager.State(r.Context())
	if e != nil {
		problem(w, 500, "state unavailable")
		return
	}
	jsonOut(w, 200, telemetry.At(time.Now(), st.Rules))
}
func (s *Server) simulate(w http.ResponseWriter, r *http.Request) {
	var p simulation.Packet
	if !decode(w, r, &p) {
		return
	}
	st, e := s.manager.State(r.Context())
	if e != nil {
		problem(w, 500, "state unavailable")
		return
	}
	out, e := simulation.Evaluate(st.Rules, p)
	if e != nil {
		problem(w, 400, e.Error())
		return
	}
	jsonOut(w, 200, out)
}
func (s *Server) listKeys(w http.ResponseWriter, r *http.Request) {
	if s.keys == nil {
		problem(w, 503, "key store unavailable")
		return
	}
	keys, e := s.keys.Keys(r.Context())
	if e != nil {
		problem(w, 500, "key list unavailable")
		return
	}
	jsonOut(w, 200, keys)
}
func (s *Server) issueKey(w http.ResponseWriter, r *http.Request) {
	if s.keys == nil {
		problem(w, 503, "key store unavailable")
		return
	}
	var body struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}
	if !decode(w, r, &body) {
		return
	}
	k, raw, e := auth.Issue(body.Name, body.Scopes)
	if e != nil {
		problem(w, 400, e.Error())
		return
	}
	if e = s.keys.CreateKey(r.Context(), k); e != nil {
		problem(w, 500, "key creation failed")
		return
	}
	jsonOut(w, 201, map[string]any{"key": k, "token": raw})
}
func (s *Server) revokeKey(w http.ResponseWriter, r *http.Request) {
	if s.keys == nil {
		problem(w, 503, "key store unavailable")
		return
	}
	if e := s.keys.RevokeKey(r.Context(), r.PathValue("id")); e != nil {
		problem(w, 404, "key not found or already revoked")
		return
	}
	jsonOut(w, 200, map[string]string{"status": "revoked"})
}
func (s *Server) bearer(w http.ResponseWriter, r *http.Request, scope string) bool {
	if s.keys == nil {
		problem(w, 503, "key store unavailable")
		return false
	}
	v := r.Header.Get("Authorization")
	if !strings.HasPrefix(v, "Bearer ") || len(v) != 74 {
		problem(w, 401, "valid bearer required")
		return false
	}
	k, e := s.keys.KeyByHash(r.Context(), auth.Hash(strings.TrimPrefix(v, "Bearer ")))
	if e != nil || !auth.Allows(k, scope, time.Now()) {
		problem(w, 403, "key invalid, expired, revoked or missing scope")
		return false
	}
	return true
}
func (s *Server) integrationStats(w http.ResponseWriter, r *http.Request) {
	if s.bearer(w, r, auth.Read) {
		s.stats(w, r)
	}
}
func (s *Server) integrationSample(w http.ResponseWriter, r *http.Request) {
	if s.bearer(w, r, auth.Read) {
		s.sample(w, r)
	}
}
func (s *Server) integrationSimulate(w http.ResponseWriter, r *http.Request) {
	if s.bearer(w, r, auth.Simulate) {
		if r.Header.Get("Content-Type") != "application/json" {
			problem(w, 415, "application/json required")
			return
		}
		s.simulate(w, r)
	}
}

// stream sends bounded synthetic telemetry bursts. EventSource reconnects after EOF.
func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	select {
	case s.streams <- struct{}{}:
		defer func() { <-s.streams }()
	default:
		problem(w, 429, "too many streams")
		return
	}
	f, ok := w.(http.Flusher)
	if !ok {
		problem(w, 500, "stream unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	end := time.NewTimer(10 * time.Second)
	defer end.Stop()
	for {
		st, e := s.manager.State(r.Context())
		if e != nil {
			return
		}
		b, e := json.Marshal(telemetry.At(time.Now(), st.Rules))
		if e != nil {
			return
		}
		if _, e = w.Write(append(append([]byte("data: "), b...), '\n', '\n')); e != nil {
			return
		}
		f.Flush()
		select {
		case <-r.Context().Done():
			return
		case <-end.C:
			return
		case <-ticker.C:
		}
	}
}
