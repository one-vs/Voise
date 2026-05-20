package native

import (
	"context"
	"encoding/json"
	"voise/internal/voiceagent/voicelog"
)

func CheckCalendarSlot(ctx context.Context, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("check_calendar_slot tool called")
	return map[string]interface{}{"available": true}, nil
}

func ListAvailableSlots(ctx context.Context, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("list_available_slots tool called")
	return []map[string]string{
		{"datetime": "2023-10-27T10:00:00Z", "master_id": "master1"},
		{"datetime": "2023-10-27T11:00:00Z", "master_id": "master1"},
	}, nil
}

func BookAppointment(ctx context.Context, args json.RawMessage) (interface{}, error) {
	l := voicelog.FromContext(ctx)
	l.Info().Msg("book_appointment tool called")
	return map[string]string{"appointment_id": "123", "status": "confirmed"}, nil
}
