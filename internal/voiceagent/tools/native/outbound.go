package native

import (
	"context"
	"encoding/json"
	"voise/internal/voiceagent/voicelog"
)

func OutboundCall(ctx context.Context, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("outbound_call tool called")
	return map[string]string{"status": "queued"}, nil
}
