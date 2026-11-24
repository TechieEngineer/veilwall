# Security policy

The integrated beta is a development preview, not a firewall that protects a host. It must not replace existing security controls. Do not expose the dashboard through a reverse proxy, tunnel or public port. It trusts local processes and is not multi-user authenticated. The scoped bearer keys authorize local integration routes only. The per-process mutation token plus exact Host/Origin checks mitigate browser cross-origin requests; they do not protect from other local users or malicious software. Do not run demo mode as root or grant container network administration capabilities. Only use a private directory for SQLite; do not commit database files or tokens.

Report security issues privately through GitHub private vulnerability reporting (enable it in repository settings before publishing); do not publish sensitive proof-of-concept information in public issues. Maintainers should establish supported versions and a disclosure process before production releases.
