package simulation

import (
	"github.com/veilwall/veilwall/internal/models"
	"testing"
)

func TestShadowAndPriority(t *testing.T) {
	base := models.Rule{ID: "shadow", Name: "shadow", Direction: "input", Protocol: "tcp", Source: "192.0.2.0/24", Port: 443, Action: "drop", Mode: "shadow", Priority: 1}
	active := base
	active.ID = "active"
	active.Mode = "active"
	active.Priority = 2
	p := Packet{"input", "tcp", "192.0.2.10", "198.51.100.1", 443}
	r, e := Evaluate([]models.Rule{active, base}, p)
	if e != nil || r.Verdict != "drop" || len(r.ShadowMatches) != 1 || !r.Simulated {
		t.Fatal(r, e)
	}
	r, e = Evaluate([]models.Rule{base}, p)
	if e != nil || r.Verdict != "accept" {
		t.Fatal(r, e)
	}
	p.Source = "203.0.113.4"
	r, e = Evaluate([]models.Rule{active, base}, p)
	if e != nil || r.Verdict != "accept" || len(r.ShadowMatches) != 0 {
		t.Fatal(r, e)
	}
}
