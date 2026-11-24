//go:build !linux

package firewall

import (
	"context"
	"errors"
)

func NewLinuxEngine(context.Context) (Engine, error) {
	return nil, errors.New("native validation requires Linux; use --demo")
}
