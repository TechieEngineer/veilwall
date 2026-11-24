# Verification — 1.0.0-beta.1

## Passed in the packaging environment

- `go test -race ./internal/{models,simulation,auth,telemetry,firewall,api}` — packages with tests passed, including rule rollback, stale revisions, API origin guards, scoped bearer issuance, scope isolation and revocation using a test store, and synthetic policy matching.
- `go vet` on the same dependency-free Go packages — passed.
- `npm --prefix web test` — four dependency-free browser-evaluator tests passed (shadow semantics, active priority, IPv6, malformed input).
- TypeScript transpiler syntax checks on both JSX source files — passed; this is not a full Vite build.
- Python SQLite engine parsed and executed the existing schema — passed; this is not a Go SQLite integration test.
- YAML parsing for workflows, Compose and OpenAPI — passed; parsing does not verify deployment.
- All Go packages type-compiled against a temporary empty SQLite import stub, with tests disabled — passed. The stub is not part of this distribution, is not a database driver, and cannot validate SQLite runtime behavior.

## Not verified here

Network access to Go and npm dependency servers was unavailable. No lockfiles were generated. Full build with the actual SQLite driver, persistent-key tests, Vite production build, end-to-end browser testing, Docker/systemd runtime, GitHub Pages and release jobs still need verification. The user reported that the v0.1 frontend built successfully; that does not verify this beta. Native packet enforcement is deliberately absent.

Run on your machine before publishing:

```powershell
go mod tidy
npm.cmd --prefix web install
npm.cmd --prefix web test
npm.cmd --prefix web run build
go vet ./...
go test ./... -v
go run ./cmd/veilwall --demo --db=veilwall.db
```

Then verify sample-policy stage/confirm/expiry, editing and JSON round-trip, simulated traffic and shadow matches, SSE reconnect, API key issuance/read/revocation, restart persistence, and the browser-only preview. Do not expose the service publicly.
