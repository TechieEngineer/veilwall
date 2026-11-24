package database

import (
	"context"
	"database/sql"
	_ "embed"
	"github.com/veilwall/veilwall/internal/models"
	_ "modernc.org/sqlite"
	"time"
)

//go:embed migrations/001_init.sql
var schema string

type DB struct{ *sql.DB }

func Open(ctx context.Context, path string) (*DB, error) {
	d, e := sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}
	d.SetMaxOpenConns(1)
	d.SetMaxIdleConns(1)
	fail := func(e error) (*DB, error) { d.Close(); return nil, e }
	for _, q := range []string{"PRAGMA busy_timeout=5000", "PRAGMA foreign_keys=ON", "PRAGMA journal_mode=WAL"} {
		if _, e = d.ExecContext(ctx, q); e != nil {
			return fail(e)
		}
	}
	tx, e := d.BeginTx(ctx, nil)
	if e != nil {
		return fail(e)
	}
	if _, e = tx.ExecContext(ctx, schema); e != nil {
		tx.Rollback()
		return fail(e)
	}
	if e = tx.Commit(); e != nil {
		return fail(e)
	}
	return &DB{d}, nil
}
func (d *DB) Rules(ctx context.Context) ([]models.Rule, error) {
	rs, e := d.QueryContext(ctx, `SELECT id,name,direction,protocol,source,destination,port,action,mode,priority FROM rules ORDER BY priority,id`)
	if e != nil {
		return nil, e
	}
	defer rs.Close()
	out := []models.Rule{}
	for rs.Next() {
		var r models.Rule
		if e = rs.Scan(&r.ID, &r.Name, &r.Direction, &r.Protocol, &r.Source, &r.Destination, &r.Port, &r.Action, &r.Mode, &r.Priority); e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, rs.Err()
}
func (d *DB) Commit(ctx context.Context, rules []models.Rule, id string) error {
	if e := models.ValidateRules(rules); e != nil {
		return e
	}
	tx, e := d.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "DELETE FROM rules"); e != nil {
		return e
	}
	for _, r := range rules {
		if _, e = tx.ExecContext(ctx, `INSERT INTO rules(id,name,direction,protocol,source,destination,port,action,mode,priority) VALUES (?,?,?,?,?,?,?,?,?,?)`, r.ID, r.Name, r.Direction, r.Protocol, r.Source, r.Destination, r.Port, r.Action, r.Mode, r.Priority); e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO audit_logs(action,actor,transaction_id,details) VALUES ('confirmed','local',?,'simulated rules persisted')`, id); e != nil {
		return e
	}
	return tx.Commit()
}
func (d *DB) Audit(ctx context.Context, action, id, details string) error {
	_, e := d.ExecContext(ctx, `INSERT INTO audit_logs(action,actor,transaction_id,details) VALUES (?,'local',?,?)`, action, id, details)
	return e
}
func (d *DB) Logs(ctx context.Context) ([]models.AuditLog, error) {
	rs, e := d.QueryContext(ctx, `SELECT id,action,actor,transaction_id,details,success,created_at FROM audit_logs ORDER BY id DESC LIMIT 100`)
	if e != nil {
		return nil, e
	}
	defer rs.Close()
	out := []models.AuditLog{}
	for rs.Next() {
		var a models.AuditLog
		var ts string
		if e = rs.Scan(&a.ID, &a.Action, &a.Actor, &a.TransactionID, &a.Details, &a.Success, &ts); e != nil {
			return nil, e
		}
		a.CreatedAt, e = time.Parse(time.RFC3339Nano, ts)
		if e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rs.Err()
}
