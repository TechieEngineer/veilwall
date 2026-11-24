// Package simulation evaluates synthetic packets without accessing a network.
package simulation

import (
	"errors"
	"github.com/veilwall/veilwall/internal/models"
	"net/netip"
	"sort"
)

type Packet struct {
	Direction   string `json:"direction"`
	Protocol    string `json:"protocol"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Port        uint16 `json:"port"`
}
type Result struct {
	Verdict       string   `json:"verdict"`
	MatchedRule   string   `json:"matched_rule,omitempty"`
	ShadowMatches []string `json:"shadow_matches"`
	Simulated     bool     `json:"simulated"`
}

func matchesAddr(rule string, addr netip.Addr) bool {
	if rule == "" {
		return true
	}
	if a, e := netip.ParseAddr(rule); e == nil {
		return a == addr
	}
	p, e := netip.ParsePrefix(rule)
	return e == nil && p.Contains(addr)
}
func Evaluate(rules []models.Rule, p Packet) (Result, error) {
	result := Result{Verdict: "accept", ShadowMatches: []string{}, Simulated: true}
	if err := models.ValidateRules(rules); err != nil {
		return result, err
	}
	src, e := netip.ParseAddr(p.Source)
	if e != nil || src.Zone() != "" {
		return result, errors.New("invalid source IP")
	}
	dst, e := netip.ParseAddr(p.Destination)
	if e != nil || dst.Zone() != "" || src.Is4() != dst.Is4() {
		return result, errors.New("invalid destination IP or mixed families")
	}
	switch p.Direction {
	case "input", "output", "forward":
	default:
		return result, errors.New("invalid direction")
	}
	switch p.Protocol {
	case "tcp", "udp":
	case "icmp", "icmpv6":
		if p.Port != 0 || (p.Protocol == "icmp") != src.Is4() {
			return result, errors.New("invalid ICMP packet")
		}
	default:
		return result, errors.New("invalid protocol")
	}
	ordered := append([]models.Rule{}, rules...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Priority == ordered[j].Priority {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].Priority < ordered[j].Priority
	})
	for _, r := range ordered {
		if r.Mode == "disabled" || r.Direction != p.Direction || (r.Protocol != "any" && r.Protocol != p.Protocol) || (r.Port != 0 && r.Port != p.Port) || !matchesAddr(r.Source, src) || !matchesAddr(r.Destination, dst) {
			continue
		}
		if r.Mode == "shadow" {
			result.ShadowMatches = append(result.ShadowMatches, r.ID)
			continue
		}
		result.Verdict = r.Action
		result.MatchedRule = r.ID
		break
	}
	return result, nil
}
