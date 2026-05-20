package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"voise/internal/voiceagent/voicelog"
)

func SummarizeCallJob(ctx context.Context, db *sqlx.DB, callID uuid.UUID) {
	voicelog.Logger.Info().Str("call_id", callID.String()).Msg("Starting post-call summarization")

	if db == nil {
		return
	}

	// 1. Fetch transcript
	var transcripts []struct {
		Speaker string `db:"speaker"`
		Content string `db:"content"`
	}
	err := db.SelectContext(ctx, &transcripts, "SELECT speaker, content FROM call_transcripts WHERE call_id = $1 ORDER BY ts_offset_ms", callID)
	if err != nil {
		voicelog.Logger.Error().Err(err).Msg("Failed to fetch transcripts for summarization")
		return
	}

	// 2. Mock call to Gemini Text API (as it requires a separate client/setup)
	summary := "Summarization not fully implemented. Call had " + string(rune(len(transcripts))) + " turns."

	// 3. Save agent_summary to calls table
	_, err = db.ExecContext(ctx, "UPDATE calls SET agent_summary = $1, status = 'completed' WHERE id = $2", summary, callID)
	if err != nil {
		voicelog.Logger.Error().Err(err).Msg("Failed to update call summary")
	}
}
