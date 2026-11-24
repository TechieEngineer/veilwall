CREATE TABLE IF NOT EXISTS rules (
 id TEXT PRIMARY KEY, name TEXT NOT NULL,
 direction TEXT NOT NULL CHECK(direction IN ('input','output','forward')),
 protocol TEXT NOT NULL CHECK(protocol IN ('any','tcp','udp','icmp','icmpv6')),
 source TEXT NOT NULL DEFAULT '', destination TEXT NOT NULL DEFAULT '',
 port INTEGER NOT NULL CHECK(port BETWEEN 0 AND 65535),
 action TEXT NOT NULL CHECK(action IN ('accept','drop','reject')),
 mode TEXT NOT NULL DEFAULT 'disabled' CHECK(mode IN ('active','shadow','disabled')),
 priority INTEGER NOT NULL CHECK(priority BETWEEN 0 AND 10000)
);
CREATE INDEX IF NOT EXISTS rules_priority ON rules(priority,id);
CREATE TABLE IF NOT EXISTS audit_logs (
 id INTEGER PRIMARY KEY, action TEXT NOT NULL, actor TEXT NOT NULL,
 transaction_id TEXT NOT NULL, details TEXT NOT NULL DEFAULT '', success INTEGER NOT NULL DEFAULT 1 CHECK(success IN (0,1)),
 created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE TABLE IF NOT EXISTS api_keys (
 id TEXT PRIMARY KEY, name TEXT NOT NULL, prefix TEXT NOT NULL,
 key_hash TEXT NOT NULL UNIQUE, scopes TEXT NOT NULL DEFAULT '[]',
 created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP, expires_at TEXT, revoked INTEGER NOT NULL DEFAULT 0 CHECK(revoked IN (0,1))
);
CREATE TABLE IF NOT EXISTS system_config (
 key TEXT PRIMARY KEY,value TEXT NOT NULL,updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
INSERT OR IGNORE INTO system_config(key,value) VALUES ('schema_version','1'),('rollback_timeout_seconds','30'),('enforcement','false');
