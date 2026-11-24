// Package telemetry emits synthetic samples. No packets or processes are inspected.
package telemetry

import (
	"github.com/veilwall/veilwall/internal/models"
	"github.com/veilwall/veilwall/internal/simulation"
	"time"
)

type Sample struct {
	Time      time.Time         `json:"time"`
	Simulated bool              `json:"simulated"`
	Packet    simulation.Packet `json:"packet"`
	Result    simulation.Result `json:"result"`
	Bytes     int               `json:"bytes"`
	Process   string            `json:"process"`
}

func At(t time.Time, rules []models.Rule) Sample {
	n := int(t.Unix() % 4)
	ports := []uint16{443, 80, 53, 22}
	protos := []string{"tcp", "tcp", "udp", "tcp"}
	processes := []string{"demo-web", "demo-http", "demo-dns", "demo-ssh"}
	sources := []string{"192.0.2.10", "198.51.100.8", "203.0.113.12", "192.0.2.20"}
	if n < 0 {
		n += 4
	}
	p := simulation.Packet{Direction: "input", Protocol: protos[n], Source: sources[n], Destination: "198.51.100.1", Port: ports[n]}
	r, _ := simulation.Evaluate(rules, p)
	return Sample{t.UTC(), true, p, r, 256 + n*384, processes[n]}
}
