package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Call struct {
	ID            uuid.UUID `db:"id"`
	TenantID      uuid.UUID `db:"tenant_id"`
	LocationID    uuid.UUID `db:"location_id"`
	TwilioCallSID string    `db:"twilio_call_sid"`
	Direction     string    `db:"direction"`
	FromNumber    string    `db:"from_number"`
	ToNumber      string    `db:"to_number"`
	Status        string    `db:"status"`
	StartedAt     time.Time `db:"started_at"`
	CreatedAt     time.Time `db:"created_at"`
}

func SaveCall(ctx context.Context, db *sqlx.DB, call *Call) error {
	query := `INSERT INTO calls (id, tenant_id, location_id, twilio_call_sid, direction, from_number, to_number, status, started_at)
	          VALUES (:id, :tenant_id, :location_id, :twilio_call_sid, :direction, :from_number, :to_number, :status, :started_at)`
	_, err := db.NamedExecContext(ctx, query, call)
	return err
}

func SaveTranscript(ctx context.Context, db *sqlx.DB, callID uuid.UUID, speaker, content string, offsetMs int) error {
	query := `INSERT INTO call_transcripts (id, call_id, speaker, content, ts_offset_ms)
	          VALUES ($1, $2, $3, $4, $5)`
	_, err := db.ExecContext(ctx, query, uuid.New(), callID, speaker, content, offsetMs)
	return err
}
