package pg

import (
	"context"
	"fmt"
	"time"

	"github.com/ravikumar/mongodb-inspector/internal/domain"
)

type ConnectionStore struct {
	db DBTX
}

func NewConnectionStore(db DBTX) *ConnectionStore {
	return &ConnectionStore{db: db}
}

func (s *ConnectionStore) Create(ctx context.Context, conn *domain.Connection) error {
	now := time.Now()
	conn.CreatedAt = now
	conn.UpdatedAt = now

	err := s.db.QueryRowContext(ctx,
		`INSERT INTO connections (name, connection_string, database)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		conn.Name, conn.ConnectionString, conn.Database,
	).Scan(&conn.ID, &conn.CreatedAt, &conn.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create connection: %w", err)
	}
	return nil
}

func (s *ConnectionStore) List(ctx context.Context, offset, limit int) ([]domain.Connection, int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM connections`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count connections: %w", err)
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, connection_string, database, created_at, updated_at
		 FROM connections ORDER BY created_at DESC OFFSET $1 LIMIT $2`, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list connections: %w", err)
	}
	defer rows.Close()

	var conns []domain.Connection
	for rows.Next() {
		var c domain.Connection
		if err := rows.Scan(&c.ID, &c.Name, &c.ConnectionString, &c.Database, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan connection: %w", err)
		}
		conns = append(conns, c)
	}
	return conns, total, rows.Err()
}

func (s *ConnectionStore) Get(ctx context.Context, id string) (*domain.Connection, error) {
	var c domain.Connection
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, connection_string, database, created_at, updated_at
		 FROM connections WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.ConnectionString, &c.Database, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}
	return &c, nil
}

func (s *ConnectionStore) UpdateDatabase(ctx context.Context, id, database string) (*domain.Connection, error) {
	var c domain.Connection
	err := s.db.QueryRowContext(ctx,
		`UPDATE connections SET database = $2, updated_at = now()
		 WHERE id = $1
		 RETURNING id, name, connection_string, database, created_at, updated_at`,
		id, database,
	).Scan(&c.ID, &c.Name, &c.ConnectionString, &c.Database, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("update connection database: %w", err)
	}
	return &c, nil
}

func (s *ConnectionStore) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM connections WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("connection not found: %s", id)
	}
	return nil
}
