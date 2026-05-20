package persistence

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func LookupCustomerByPhone(ctx context.Context, db *sqlx.DB, phone string) (*uuid.UUID, error) {
	var id uuid.UUID
	query := `SELECT id FROM customers WHERE phone = $1 LIMIT 1`
	err := db.GetContext(ctx, &id, query, phone)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}
