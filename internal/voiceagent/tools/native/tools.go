package native

import (
	"context"
	"encoding/json"
	"github.com/jmoiron/sqlx"
	"voise/internal/voiceagent/voicelog"
)

func LookupCustomer(ctx context.Context, db *sqlx.DB, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("lookup_customer tool called")

	var params struct {
		Phone string `json:"phone"`
	}
	json.Unmarshal(args, &params)

	if db == nil {
		return map[string]string{"name": "John Doe (Mock)", "status": "active"}, nil
	}

	var customer struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	err := db.GetContext(ctx, &customer, "SELECT id, name FROM customers WHERE phone = $1 LIMIT 1", params.Phone)
	if err != nil {
		return nil, err
	}

	return customer, nil
}

func TransferToHuman(ctx context.Context, db *sqlx.DB, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("transfer_to_human tool called")
	return map[string]string{"status": "transferring"}, nil
}

func EndCall(ctx context.Context, db *sqlx.DB, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("end_call tool called")
	return map[string]string{"status": "ended"}, nil
}

func LogInteraction(ctx context.Context, db *sqlx.DB, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("log_interaction tool called")

	if db != nil {
		// Log note to database
	}

	return map[string]bool{"queued": true}, nil
}

func SaveNote(ctx context.Context, db *sqlx.DB, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("save_note tool called")

	if db != nil {
		// Save note to database
	}

	return map[string]bool{"queued": true}, nil
}
