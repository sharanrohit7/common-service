package db

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// DB defines the interface for database operations.
type DB interface {
	// Exec executes a query without returning rows.
	// args are for parameterized queries to prevent SQL injection.
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

	// Query executes a query that returns rows.
	// args are for parameterized queries to prevent SQL injection.
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// QueryRow executes a query that returns a single row.
	// args are for parameterized queries to prevent SQL injection.
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row

	// Prepare creates a prepared statement for later queries or executions.
	// This is the primary defense against SQL injection.
	Prepare(ctx context.Context, query string) (*sql.Stmt, error)

	// BeginTx starts a transaction.
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)

	// Close closes the database connection.
	Close() error

	// Ping checks the database connection.
	Ping(ctx context.Context) error
}

// Tx defines the interface for database transactions.
type Tx interface {
	// Exec executes a query within the transaction.
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

	// Query executes a query that returns rows within the transaction.
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)

	// QueryRow executes a query that returns a single row within the transaction.
	// args are for parameterized queries to prevent SQL injection.
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row

	// Prepare creates a prepared statement for use within the transaction.
	Prepare(ctx context.Context, query string) (*sql.Stmt, error)

	// Commit commits the transaction.
	Commit() error

	// Rollback rolls back the transaction.
	Rollback() error
}

// PostgresDB implements DB using PostgreSQL.
type PostgresDB struct {
	db *sql.DB
}

// NewPostgresDB creates a new PostgreSQL database connection.
// connectionString: PostgreSQL connection string (e.g., "postgres://user:pass@localhost/dbname?sslmode=disable")
func NewPostgresDB(connectionString string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresDB{db: db}, nil
}

// Exec executes a query without returning rows.
func (p *PostgresDB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return p.db.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows.
func (p *PostgresDB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return p.db.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns a single row.
func (p *PostgresDB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return p.db.QueryRowContext(ctx, query, args...)
}

// Prepare creates a prepared statement for later queries or executions.
func (p *PostgresDB) Prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	return p.db.PrepareContext(ctx, query)
}

// BeginTx starts a transaction.
func (p *PostgresDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	tx, err := p.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &PostgresTx{tx: tx}, nil
}

// Close closes the database connection.
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// Ping checks the database connection.
func (p *PostgresDB) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// PostgresTx implements Tx for PostgreSQL.
type PostgresTx struct {
	tx *sql.Tx
}

// Exec executes a query within the transaction.
func (p *PostgresTx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return p.tx.ExecContext(ctx, query, args...)
}

// Query executes a query that returns rows within the transaction.
func (p *PostgresTx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return p.tx.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns a single row within the transaction.
func (p *PostgresTx) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return p.tx.QueryRowContext(ctx, query, args...)
}

// Prepare creates a prepared statement for use within the transaction.
func (p *PostgresTx) Prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	return p.tx.PrepareContext(ctx, query)
}

// Commit commits the transaction.
func (p *PostgresTx) Commit() error {
	return p.tx.Commit()
}

// Rollback rolls back the transaction.
func (p *PostgresTx) Rollback() error {
	return p.tx.Rollback()
}
