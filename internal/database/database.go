package database

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"orderly/internal/metrics"
)

var (
	ErrOrderNotFound = errors.New("order not found")
)

// Order represents an order in the system.
type Order struct {
	ID     int     `json:"id"`
	Item   string  `json:"item"`
	Amount float64 `json:"amount"`
}

// DB wraps the sql.DB and provides order operations.
type DB struct {
	*sql.DB
	slowMode bool
}

// NewDB creates a new database connection pool with a deliberately small pool size.
// The connection string is read from the DATABASE_URL environment variable.
func NewDB() (*DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5433/orderly?sslmode=disable"
	}

	// Use pgx stdlib driver for better performance
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	// Deliberately small pool to demonstrate pool exhaustion
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Set max open connections metric
	metrics.DBPoolMaxOpenConnections.Set(5)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	// Initialize schema
	if err := initSchema(ctx, db); err != nil {
		return nil, err
	}

	// Check for slow mode flag
	slowMode := os.Getenv("SLOW_DB") == "true"

	return &DB{DB: db, slowMode: slowMode}, nil
}

// initSchema creates the orders table if it doesn't exist.
func initSchema(ctx context.Context, db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY,
		item TEXT NOT NULL,
		amount NUMERIC(10,2) NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`
	_, err := db.ExecContext(ctx, query)
	return err
}

// CreateOrder inserts a new order and returns it with the assigned ID.
func (db *DB) CreateOrder(ctx context.Context, item string, amount float64) (*Order, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.DBQueryDurationSeconds.WithLabelValues("insert_order").Observe(duration)
		metrics.DBQueryDurationSummarySeconds.WithLabelValues("insert_order").Observe(duration)
	}()

	// Slow mode: add artificial delay before query
	if db.slowMode {
		// Sleep for 100-500ms to simulate slow database
		time.Sleep(time.Duration(100+time.Now().UnixNano()%400) * time.Millisecond)
	}

	var order Order
	query := `INSERT INTO orders (item, amount) VALUES ($1, $2) RETURNING id, item, amount`
	err := db.QueryRowContext(ctx, query, item, amount).Scan(&order.ID, &order.Item, &order.Amount)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return nil, err
		}
		return nil, err
	}
	return &order, nil
}

// GetOrder retrieves an order by ID.
func (db *DB) GetOrder(ctx context.Context, id int) (*Order, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.DBQueryDurationSeconds.WithLabelValues("get_order").Observe(duration)
		metrics.DBQueryDurationSummarySeconds.WithLabelValues("get_order").Observe(duration)
	}()

	var order Order
	query := `SELECT id, item, amount FROM orders WHERE id = $1`
	err := db.QueryRowContext(ctx, query, id).Scan(&order.ID, &order.Item, &order.Amount)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

// UpdatePoolMetrics updates the database pool metrics from sql.DBStats.
// This should be called periodically (e.g., from a background goroutine).
func (db *DB) UpdatePoolMetrics() {
	stats := db.Stats()
	metrics.DBPoolOpenConnections.Set(float64(stats.OpenConnections))
	metrics.DBPoolInUseConnections.Set(float64(stats.InUse))
	metrics.DBPoolIdleConnections.Set(float64(stats.Idle))
	metrics.DBPoolWaitCount.Add(float64(stats.WaitCount))
	metrics.DBPoolWaitDurationSeconds.Add(stats.WaitDuration.Seconds())
}