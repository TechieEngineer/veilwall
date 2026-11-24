package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/veilwall/veilwall/internal/models"
	"time"
)

func (d *DB) CreateKey(ctx context.Context, k models.APIKey) error {
	sc, e := json.Marshal(k.Scopes)
	if e != nil {
		return e
	}
	_, e = d.ExecContext(ctx, `INSERT INTO api_keys(id,name,prefix,key_hash,scopes,created_at,expires_at,revoked) VALUES(?,?,?,?,?,?,?,0)`, k.ID, k.Name, k.Prefix, k.KeyHash, string(sc), k.CreatedAt.Format(time.RFC3339Nano), k.ExpiresAt.Format(time.RFC3339Nano))
	return e
}

type scanner interface{ Scan(...any) error }

func scanKey(s scanner) (models.APIKey, error) {
	var k models.APIKey
	var scopes, created string
	var expires sql.NullString
	e := s.Scan(&k.ID, &k.Name, &k.Prefix, &k.KeyHash, &scopes, &created, &expires, &k.Revoked)
	if e != nil {
		return k, e
	}
	if e = json.Unmarshal([]byte(scopes), &k.Scopes); e != nil {
		return k, e
	}
	k.CreatedAt, e = time.Parse(time.RFC3339Nano, created)
	if e != nil {
		return k, e
	}
	if expires.Valid {
		v, e := time.Parse(time.RFC3339Nano, expires.String)
		if e != nil {
			return k, e
		}
		k.ExpiresAt = &v
	}
	return k, nil
}

const keyCols = `id,name,prefix,key_hash,scopes,created_at,expires_at,revoked`

func (d *DB) Keys(ctx context.Context) ([]models.APIKey, error) {
	rs, e := d.QueryContext(ctx, `SELECT `+keyCols+` FROM api_keys ORDER BY created_at DESC LIMIT 100`)
	if e != nil {
		return nil, e
	}
	defer rs.Close()
	out := []models.APIKey{}
	for rs.Next() {
		k, e := scanKey(rs)
		if e != nil {
			return nil, e
		}
		out = append(out, k)
	}
	return out, rs.Err()
}
func (d *DB) KeyByHash(ctx context.Context, h string) (models.APIKey, error) {
	return scanKey(d.QueryRowContext(ctx, `SELECT `+keyCols+` FROM api_keys WHERE key_hash=?`, h))
}
func (d *DB) RevokeKey(ctx context.Context, id string) error {
	r, e := d.ExecContext(ctx, `UPDATE api_keys SET revoked=1 WHERE id=? AND revoked=0`, id)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
