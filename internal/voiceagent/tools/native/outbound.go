package native

import (
	"context"
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"voise/internal/voiceagent/voicelog"
)

func OutboundCall(ctx context.Context, db *sqlx.DB, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("outbound_call tool called")
	return map[string]string{"status": "queued"}, nil
}
