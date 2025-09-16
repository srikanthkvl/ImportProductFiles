package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomerDB manages connections to a specific customer's Postgres database.
type CustomerDB struct {
	Pool *pgxpool.Pool
}

// ConnectCustomerDB connects to the customer's Postgres using the given DSN.
func ConnectCustomerDB(ctx context.Context, dsn string) (*CustomerDB, error) {
	if dsn == "" {
		return nil, errors.New("empty customer DSN")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &CustomerDB{Pool: pool}, nil
}

// EnsureTargetTable ensures a table exists with the given name and a JSONB column named data.
func (c *CustomerDB) EnsureTargetTable(ctx context.Context, tableName string) error {
	if tableName == "" {
		return errors.New("empty table name")
	}
	// Simple validation to avoid SQL injection from tableName
	for _, r := range tableName {
		if !(r == '_' || r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z')) {
			return fmt.Errorf("invalid table name: %s", tableName)
		}
	}
	ddl := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id BIGSERIAL PRIMARY KEY,
		data JSONB NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);`, tableName)
	_, err := c.Pool.Exec(ctx, ddl)
	return err
}

// InsertJSONB inserts a row into the target table with the JSONB document.
func (c *CustomerDB) InsertJSONB(ctx context.Context, tableName string, data []byte) error {
	fmt.Println("Inserting into table:%s, values: %s", tableName, string(data))
	ddl := fmt.Sprintf("INSERT INTO %s (data) VALUES ($1)", tableName)
	_, err := c.Pool.Exec(ctx, ddl, data)
	return err
}


