#!/usr/bin/env bash
# Install an already-built, trusted local binary. No remote shell execution.
set -euo pipefail
[[ "$(uname -s)" == Linux ]] || { echo 'Linux user-service installer only'; exit 1; }
[[ "$(id -u)" != 0 ]] || { echo 'Run as a regular user, not root.'; exit 1; }
root="$(cd "$(dirname "$0")/.." && pwd)"
binary="${1:-$root/bin/veilwall}"
[[ -f "$binary" && -x "$binary" ]] || { echo 'Build first with bash scripts/setup.sh or pass a trusted binary path.'; exit 1; }
install -d -m 0700 "$HOME/.local/bin" "$HOME/.local/share/veilwall" "$HOME/.config/systemd/user"
install -m 0755 "$binary" "$HOME/.local/bin/veilwall"
install -m 0644 "$root/deploy/systemd/veilwall.service" "$HOME/.config/systemd/user/veilwall.service"
if command -v systemctl >/dev/null && systemctl --user daemon-reload; then
  systemctl --user enable --now veilwall.service
  echo 'Dashboard: http://127.0.0.1:8080'
else
  echo 'User systemd unavailable. Run ~/.local/bin/veilwall --demo manually.'
fi
