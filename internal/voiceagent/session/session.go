package session

import (
	"context"
	"encoding/base64"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
	"voise/internal/voiceagent/audio"
	"voise/internal/voiceagent/gemini"
	"voise/internal/voiceagent/persistence"
	"voise/internal/voiceagent/tools"
	"voise/internal/voiceagent/voicelog"
)

type Session struct {
	ID         uuid.UUID
	CallID     uuid.UUID
	StreamSID  string
	CustomerID *uuid.UUID
	TwilioConn *websocket.Conn
	GeminiConn *gemini.Conn
	GeminiClient *gemini.Client
	DB         *sqlx.DB
	ToolRouter *tools.ToolRouter

	JitterBuf  *audio.JitterBuffer

	mu sync.Mutex
}

func (s *Session) Run(ctx context.Context) {
	log := voicelog.FromContext(ctx)
	log.Info().Msg("Session logic running")

	if s.JitterBuf == nil {
		s.JitterBuf = audio.NewJitterBuffer()
	}

	if s.GeminiClient != nil {
		conn, err := s.GeminiClient.Connect(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to connect to Gemini")
			return
		}
		s.GeminiConn = conn
		defer s.GeminiConn.Close()

		// Send setup message
		setup := map[string]interface{}{
			"setup": gemini.NewLiveConnectConfig("gemini-2.0-flash-exp", "You are a friendly receptionist."),
		}
		s.GeminiConn.SendRaw(setup)
	}

	// Start Gemini to Twilio loop
	go s.geminiToTwilioLoop(ctx)
}

func (s *Session) geminiToTwilioLoop(ctx context.Context) {
	log := voicelog.FromContext(ctx)
	for {
		var resp gemini.GeminiResponse
		if err := s.GeminiConn.ReceiveRaw(&resp); err != nil {
			if err != io.EOF {
				log.Error().Err(err).Msg("Gemini connection closed")
			}
			return
		}
		s.HandleGeminiResponse(ctx, &resp)
	}
}

func (s *Session) HandleTwilioAudio(payload []byte) error {
	pcm16 := audio.MulawToPCM16(payload)
	resampled := audio.Resample(pcm16, 8000, 16000)
	data := audio.EncodePCM16LE(resampled)

	return s.GeminiConn.SendRaw(map[string]interface{}{
		"realtimeInput": map[string]interface{}{
			"mediaChunks": []map[string]interface{}{
				{
					"mimeType": "audio/pcm;rate=16000",
					"data":     base64.StdEncoding.EncodeToString(data),
				},
			},
		},
	})
}

func (s *Session) HandleGeminiResponse(ctx context.Context, resp *gemini.GeminiResponse) {
	if resp.RealtimeInputResponse != nil && resp.RealtimeInputResponse.Transcription != nil {
		t := resp.RealtimeInputResponse.Transcription
		if t.Text != "" && s.DB != nil {
			persistence.SaveTranscript(ctx, s.DB, s.CallID, "user", t.Text, 0)
		}
	}

	if resp.ServerContent != nil {
		if resp.ServerContent.Interrupted {
			s.JitterBuf.Clear()
			s.TwilioConn.WriteJSON(map[string]string{
				"event":     "clear",
				"streamSid": s.StreamSID,
			})
		}

		if resp.ServerContent.ModelTurn != nil {
			for _, part := range resp.ServerContent.ModelTurn.Parts {
				if part.Text != "" && s.DB != nil {
					persistence.SaveTranscript(ctx, s.DB, s.CallID, "agent", part.Text, 0)
				}
				if part.InlineData != nil {
					audioBytes, _ := base64.StdEncoding.DecodeString(part.InlineData.Data)
					pcm16 := audio.DecodePCM16LE(audioBytes)
					resampled := audio.Resample(pcm16, 24000, 8000)
					mulaw := audio.PCM16ToMulaw(resampled)

					s.TwilioConn.WriteJSON(map[string]interface{}{
						"event":     "media",
						"streamSid": s.StreamSID,
						"media": map[string]string{
							"payload": base64.StdEncoding.EncodeToString(mulaw),
						},
					})
				}
			}
		}
	}

	if resp.ToolCall != nil && s.ToolRouter != nil {
		for _, call := range resp.ToolCall.FunctionCalls {
			result, err := s.ToolRouter.Invoke(ctx, call.Name, call.Args)

			response := map[string]interface{}{
				"toolResponse": map[string]interface{}{
					"functionResponses": []map[string]interface{}{
						{
							"name":     call.Name,
							"id":       call.ID,
							"response": result,
						},
					},
				},
			}
			if err != nil {
				response["toolResponse"].(map[string]interface{})["functionResponses"].([]map[string]interface{})[0]["error"] = err.Error()
			}
			s.GeminiConn.SendRaw(response)
		}
	}
}
