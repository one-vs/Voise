package outbound

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"voise/internal/voiceagent/voicelog"
)

type QueueWorker struct {
	db        *sqlx.DB
	initiator *Initiator
}

func NewQueueWorker(db *sqlx.DB, initiator *Initiator) *QueueWorker {
	return &QueueWorker{db: db, initiator: initiator}
}

func (w *QueueWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processQueue(ctx)
		}
	}
}

func (w *QueueWorker) processQueue(ctx context.Context) {
	if w.db == nil {
		return
	}

	voicelog.Logger.Debug().Msg("Polling outbound call requests")

	var requests []struct {
		ID       string `db:"id"`
		ToNumber string `db:"to_number"`
		TenantID string `db:"tenant_id"`
	}

	err := w.db.SelectContext(ctx, &requests, "SELECT id, to_number, tenant_id FROM outbound_call_requests WHERE status = 'queued' AND scheduled_at <= NOW() LIMIT 5")
	if err != nil {
		voicelog.Logger.Error().Err(err).Msg("Failed to poll outbound requests")
		return
	}

	for _, req := range requests {
		voicelog.Logger.Info().Str("to", req.ToNumber).Msg("Initiating outbound call")

		_, err := w.db.ExecContext(ctx, "UPDATE outbound_call_requests SET status = 'processing', attempts = attempts + 1 WHERE id = $1", req.ID)
		if err != nil {
			continue
		}

		sid, err := w.initiator.InitiateCall(ctx, "SYSTEM_NUMBER", req.ToNumber, "https://example.com/webhooks/twilio/voice/outbound")
		if err != nil {
			voicelog.Logger.Error().Err(err).Str("id", req.ID).Msg("Failed to initiate call")
			w.db.ExecContext(ctx, "UPDATE outbound_call_requests SET status = 'failed', last_error = $1 WHERE id = $2", err.Error(), req.ID)
			continue
		}

		w.db.ExecContext(ctx, "UPDATE outbound_call_requests SET status = 'completed', call_id = $1 WHERE id = $2", sid, req.ID)
	}
}
