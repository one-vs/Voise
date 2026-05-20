package native

import (
	"context"
	"encoding/json"
	"voise/internal/voiceagent/voicelog"
)

func LookupCustomer(ctx context.Context, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("lookup_customer tool called")
	// Implementation would go here
	return map[string]string{"name": "John Doe", "status": "active"}, nil
}

func TransferToHuman(ctx context.Context, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("transfer_to_human tool called")
	return map[string]string{"status": "transferring"}, nil
}

func EndCall(ctx context.Context, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("end_call tool called")
	return map[string]string{"status": "ended"}, nil
}
