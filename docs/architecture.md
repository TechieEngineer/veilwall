# Architecture — integrated beta

The Go daemon is the single owner of simulated rules. `internal/firewall` owns the engine interface, serialized revisioned transactions and timeout-based rollback. SQLite stores confirmed rules, audit records and hashed scoped credentials; staged state stays in memory. On a restart the last confirmed rules are loaded. This is not kernel crash recovery. The Linux adapter uses a fixed `nft --check` transaction only; it never changes host networking.

`internal/simulation` evaluates synthetic IPv4/IPv6 packets: ascending priority then ID; disabled rules are skipped; shadow matches do not change the verdict; first matching active rule terminates evaluation; unmatched defaults to accept. No traffic is sent. `internal/telemetry` generates deterministic documentation-address samples with fictional process labels. The API exposes snapshot and bounded SSE (8 concurrent streams, 10-second bursts, auto reconnect). The frontend retains 20 samples.

The local HTTP API enforces loopback peers, exact Host and same-origin headers. The ephemeral dashboard token guards mutations against cross-origin browser requests; it is not user authentication. Persistent bearer credentials authorize only designated local integration endpoints and never rule mutation. SQLite stores SHA-256 hashes of randomly generated secrets, with a 30-day expiry and revocation. Local programs still have access to the dashboard API; this is not a multi-tenant boundary.

Vite compiles the React UI into web/dist, embedded by Go. Pages runs the separate JS policy evaluator and transaction adapter entirely in-browser. Rules and preview audit state reset on reload; real keys are disabled. CI builds both code paths and tests the evaluators.

Native enforcement, live conntrack/process monitoring, threat feeds, remote access, WebSockets and alert webhooks remain outside this beta. No firewall protection is provided.
