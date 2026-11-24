package models

import (
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strings"
	"time"
)

type Rule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Direction   string `json:"direction"`
	Protocol    string `json:"protocol"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Port        uint16 `json:"port"`
	Action      string `json:"action"`
	Mode        string `json:"mode"`
	Priority    int    `json:"priority"`
}
type AuditLog struct {
	ID            int64     `json:"id"`
	Action        string    `json:"action"`
	Actor         string    `json:"actor"`
	TransactionID string    `json:"transaction_id"`
	Details       string    `json:"details"`
	Success       bool      `json:"success"`
	CreatedAt     time.Time `json:"created_at"`
}
type APIKey struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Prefix    string     `json:"prefix"`
	KeyHash   string     `json:"-"`
	Scopes    []string   `json:"scopes"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
	Revoked   bool       `json:"revoked"`
}
type SystemConfig struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

var idPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}
func validAddr(s string) bool {
	if s == "" {
		return true
	}
	if a, e := netip.ParseAddr(s); e == nil {
		return a.Zone() == ""
	}
	_, e := netip.ParsePrefix(s)
	return e == nil
}
func ValidateRules(rules []Rule) error {
	if len(rules) > 500 {
		return errors.New("maximum 500 rules")
	}
	seen := map[string]bool{}
	for i, r := range rules {
		bad := func(msg string) error { return fmt.Errorf("rule %d: %s", i+1, msg) }
		if !idPattern.MatchString(r.ID) || seen[r.ID] {
			return bad("invalid or duplicate ID")
		}
		seen[r.ID] = true
		if strings.TrimSpace(r.Name) == "" || len(r.Name) > 100 {
			return bad("name must be 1–100 bytes")
		}
		if !oneOf(r.Direction, "input", "output", "forward") || !oneOf(r.Protocol, "any", "tcp", "udp", "icmp", "icmpv6") || !oneOf(r.Action, "accept", "drop", "reject") || !oneOf(r.Mode, "active", "shadow", "disabled") {
			return bad("invalid enum")
		}
		if !validAddr(r.Source) || !validAddr(r.Destination) {
			return bad("invalid IP or CIDR")
		}
		if r.Port != 0 && !oneOf(r.Protocol, "tcp", "udp") {
			return bad("port requires TCP or UDP")
		}
		if r.Priority < 0 || r.Priority > 10000 {
			return bad("priority outside 0–10000")
		}
	}
	return nil
}
