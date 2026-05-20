package native

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"voise/internal/voiceagent/voicelog"
)

func CheckCalendarSlot(ctx context.Context, db *sqlx.DB, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("check_calendar_slot tool called")
	return map[string]interface{}{"available": true}, nil
}

func ListAvailableSlots(ctx context.Context, db *sqlx.DB, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("list_available_slots tool called")
	return []map[string]string{
		{"datetime": "2023-10-27T10:00:00Z", "master_id": "master1"},
		{"datetime": "2023-10-27T11:00:00Z", "master_id": "master1"},
	}, nil
}

func BookAppointment(ctx context.Context, db *sqlx.DB, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("book_appointment tool called")

	var params struct {
		CustomerID string `json:"customer_id"`
		ServiceID  string `json:"service_id"`
		DateTime   string `json:"datetime"`
	}
	json.Unmarshal(args, &params)

	if db == nil {
		return map[string]string{"appointment_id": "mock-123", "status": "confirmed"}, nil
	}

	id := uuid.New().String()
	_, err := db.ExecContext(ctx, "INSERT INTO appointments (id, customer_id, service_id, scheduled_at) VALUES ($1, $2, $3, $4)",
		id, params.CustomerID, params.ServiceID, params.DateTime)
	if err != nil {
		return nil, err
	}

	return map[string]string{"appointment_id": id, "status": "confirmed"}, nil
}

func CancelAppointment(ctx context.Context, db *sqlx.DB, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("cancel_appointment tool called")

	var params struct {
		ID string `json:"appointment_id"`
	}
	json.Unmarshal(args, &params)

	if db != nil {
		_, err := db.ExecContext(ctx, "UPDATE appointments SET status = 'cancelled' WHERE id = $1", params.ID)
		if err != nil {
			return nil, err
		}
	}

	return map[string]string{"status": "cancelled"}, nil
}

func RescheduleAppointment(ctx context.Context, db *sqlx.DB, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("reschedule_appointment tool called")

	var params struct {
		ID       string `json:"appointment_id"`
		DateTime string `json:"datetime"`
	}
	json.Unmarshal(args, &params)

	if db != nil {
		_, err := db.ExecContext(ctx, "UPDATE appointments SET scheduled_at = $1 WHERE id = $2", params.DateTime, params.ID)
		if err != nil {
			return nil, err
		}
	}

	return map[string]string{"appointment_id": params.ID, "status": "rescheduled"}, nil
}
