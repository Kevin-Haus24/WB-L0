package db

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"L0/internal/model"
	"L0/migrations"

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
	_, err := db.pool.Exec(ctx, migrations.CreateTables)
	return err
}

func (db *DB) SaveOrder(ctx context.Context, order model.Order, raw json.RawMessage) error {
	dateCreated, err := time.Parse(time.RFC3339, order.DateCreated)
	if err != nil {
		dateCreated = time.Now().UTC()
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
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
	if err != nil {
		return err
	}

	if err := db.saveDelivery(ctx, tx, order); err != nil {
		return err
	}
	if err := db.savePayment(ctx, tx, order); err != nil {
		return err
	}
	if err := db.saveItems(ctx, tx, order); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (db *DB) saveDelivery(ctx context.Context, tx pgx.Tx, order model.Order) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO deliveries (
			order_uid,
			name,
			phone,
			zip,
			city,
			address,
			region,
			email
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) ON CONFLICT (order_uid) DO UPDATE SET
			name = EXCLUDED.name,
			phone = EXCLUDED.phone,
			zip = EXCLUDED.zip,
			city = EXCLUDED.city,
			address = EXCLUDED.address,
			region = EXCLUDED.region,
			email = EXCLUDED.email`,
		order.OrderUID,
		order.Delivery.Name,
		order.Delivery.Phone,
		order.Delivery.Zip,
		order.Delivery.City,
		order.Delivery.Address,
		order.Delivery.Region,
		order.Delivery.Email,
	)
	return err
}

func (db *DB) savePayment(ctx context.Context, tx pgx.Tx, order model.Order) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO payments (
			order_uid,
			transaction,
			request_id,
			currency,
			provider,
			amount,
			payment_dt,
			bank,
			delivery_cost,
			goods_total,
			custom_fee
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		) ON CONFLICT (order_uid) DO UPDATE SET
			transaction = EXCLUDED.transaction,
			request_id = EXCLUDED.request_id,
			currency = EXCLUDED.currency,
			provider = EXCLUDED.provider,
			amount = EXCLUDED.amount,
			payment_dt = EXCLUDED.payment_dt,
			bank = EXCLUDED.bank,
			delivery_cost = EXCLUDED.delivery_cost,
			goods_total = EXCLUDED.goods_total,
			custom_fee = EXCLUDED.custom_fee`,
		order.OrderUID,
		order.Payment.Transaction,
		order.Payment.RequestID,
		order.Payment.Currency,
		order.Payment.Provider,
		order.Payment.Amount,
		order.Payment.PaymentDT,
		order.Payment.Bank,
		order.Payment.DeliveryCost,
		order.Payment.GoodsTotal,
		order.Payment.CustomFee,
	)
	return err
}

func (db *DB) saveItems(ctx context.Context, tx pgx.Tx, order model.Order) error {
	if _, err := tx.Exec(ctx, `DELETE FROM items WHERE order_uid = $1`, order.OrderUID); err != nil {
		return err
	}

	for _, item := range order.Items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO items (
				order_uid,
				chrt_id,
				track_number,
				price,
				rid,
				name,
				sale,
				size,
				total_price,
				nm_id,
				brand,
				status
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
			)`,
			order.OrderUID,
			item.ChrtID,
			item.TrackNumber,
			item.Price,
			item.Rid,
			item.Name,
			item.Sale,
			item.Size,
			item.TotalPrice,
			item.NmID,
			item.Brand,
			item.Status,
		); err != nil {
			return err
		}
	}

	return nil
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
