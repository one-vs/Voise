package native

import (
	"context"
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"voise/internal/voiceagent/voicelog"
)

func QueryMemory(ctx context.Context, db *sqlx.DB, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("query_memory tool called")
	return map[string]string{"summary": "Known customer with a history of booking spa treatments."}, nil
}
