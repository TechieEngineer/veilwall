//go:build linux

package firewall

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// LinuxEngine intentionally never commits nftables changes in v0.1.
type LinuxEngine struct{ *MockEngine }

func (e *LinuxEngine) Name() string { return "linux-validation" }
func NewLinuxEngine(ctx context.Context) (Engine, error) {
	path, e := exec.LookPath("nft")
	if e != nil {
		return nil, fmt.Errorf("nft not found: %w", e)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--check", "-f", "-")
	cmd.Stdin = strings.NewReader("add table inet veilwall_v01_validation\nadd chain inet veilwall_v01_validation staging\n")
	if out, e := cmd.CombinedOutput(); e != nil {
		return nil, fmt.Errorf("nft check failed (permissions or installation): %w: %s", e, out)
	}
	return &LinuxEngine{NewMockEngine()}, nil
}
