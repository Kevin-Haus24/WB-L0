package db

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"L0/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

func New(conn string) (*DB, error) {
	pool, err := pgxpool.New(context.Background(), conn)
	if err != nil {
		return nil, err
	}
	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) EnsureSchema(ctx context.Context) error {
	_, err := db.pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS orders (
		order_uid TEXT PRIMARY KEY,
		track_number TEXT,
		entry TEXT,
		locale TEXT,
		internal_signature TEXT,
		customer_id TEXT,
		delivery_service TEXT,
		shardkey TEXT,
		sm_id INT,
		date_created TIMESTAMPTZ,
		oof_shard TEXT,
		raw JSONB NOT NULL,
		created_at TIMESTAMPTZ DEFAULT now()
	)`)
	return err
}

func (db *DB) SaveOrder(ctx context.Context, order model.Order, raw json.RawMessage) error {
	dateCreated, err := time.Parse(time.RFC3339, order.DateCreated)
	if err != nil {
		dateCreated = time.Now().UTC()
	}

	_, err = db.pool.Exec(ctx,
		`INSERT INTO orders (
			order_uid,
			track_number,
			entry,
			locale,
			internal_signature,
			customer_id,
			delivery_service,
			shardkey,
			sm_id,
			date_created,
			oof_shard,
			raw
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) ON CONFLICT (order_uid) DO UPDATE SET
			track_number = EXCLUDED.track_number,
			entry = EXCLUDED.entry,
			locale = EXCLUDED.locale,
			internal_signature = EXCLUDED.internal_signature,
			customer_id = EXCLUDED.customer_id,
			delivery_service = EXCLUDED.delivery_service,
			shardkey = EXCLUDED.shardkey,
			sm_id = EXCLUDED.sm_id,
			date_created = EXCLUDED.date_created,
			oof_shard = EXCLUDED.oof_shard,
			raw = EXCLUDED.raw`,
		order.OrderUID,
		order.TrackNumber,
		order.Entry,
		order.Locale,
		order.InternalSignature,
		order.CustomerID,
		order.DeliveryService,
		order.ShardKey,
		order.SmID,
		dateCreated,
		order.OofShard,
		raw,
	)
	return err
}

func (db *DB) GetAllOrders(ctx context.Context) (map[string]json.RawMessage, error) {
	rows, err := db.pool.Query(ctx, `SELECT order_uid, raw FROM orders`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	data := make(map[string]json.RawMessage)
	for rows.Next() {
		var id string
		var raw json.RawMessage
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, err
		}
		data[id] = raw
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return data, nil
}

func (db *DB) GetOrder(ctx context.Context, orderUID string) (json.RawMessage, error) {
	var raw json.RawMessage
	err := db.pool.QueryRow(ctx, `SELECT raw FROM orders WHERE order_uid = $1`, orderUID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return raw, err
}
