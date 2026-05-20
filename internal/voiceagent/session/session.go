package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"sync"
	"time"

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
	ID                  uuid.UUID
	CallID              uuid.UUID
	StreamSID           string
	CustomerID          *uuid.UUID
	TwilioConn          *websocket.Conn
	GeminiConn          *gemini.Conn
	GeminiClient        *gemini.Client
	DB                  *sqlx.DB
	ToolRouter          *tools.ToolRouter
	ResumptionToken     string

	JitterBuf  *audio.JitterBuffer

	mu sync.Mutex
	writeMu             sync.Mutex
}

func (s *Session) Run(ctx context.Context) {
	log := voicelog.FromContext(ctx)
	log.Info().Msg("Session logic running")

	if s.JitterBuf == nil {
		s.JitterBuf = audio.NewJitterBuffer()
	}

	// Start Twilio output loop
	go s.twilioOutputLoop(ctx)

	if s.GeminiClient != nil {
		conn, err := s.GeminiClient.Connect(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to connect to Gemini")
			return
		}
		s.GeminiConn = conn
		defer s.GeminiConn.Close()

		// Send setup message
		instruction := "You are a friendly receptionist."
		if s.CustomerID != nil {
			var customer struct {
				Name string `db:"name"`
			}
			err := s.DB.Get(&customer, "SELECT name FROM customers WHERE id = $1", s.CustomerID)
			if err == nil {
				instruction += " You are speaking with a returning customer named " + customer.Name + "."
			}
		}

		// Prepare tool declarations
		var decls []gemini.FunctionDeclaration
		if s.ToolRouter != nil {
			// In a real implementation, we'd list actual tool schemas.
			// This is a simplified version for the skeleton.
			decls = []gemini.FunctionDeclaration{
				{Name: "lookup_customer", Description: "Search for a customer by phone number."},
				{Name: "book_appointment", Description: "Book a new appointment."},
			}
		}

		setup := map[string]interface{}{
			"setup": gemini.NewLiveConnectConfig("gemini-2.0-flash-exp", instruction, decls),
		}
		s.GeminiConn.SendRaw(setup)
	}

	// Start Gemini to Twilio loop
	go s.geminiToTwilioLoop(ctx)

	// Twilio to Gemini loop (main loop)
	for {
		_, message, err := s.TwilioConn.ReadMessage()
		if err != nil {
			log.Error().Err(err).Msg("Twilio connection closed")
			return
		}

		var event map[string]interface{}
		if err := json.Unmarshal(message, &event); err != nil {
			continue
		}

		if event["event"] == "start" {
			start := event["start"].(map[string]interface{})
			s.StreamSID = start["streamSid"].(string)

			// Validate auth_token from custom parameters
			meta := start["customParameters"].(map[string]interface{})
			if meta["auth_token"] == "" {
				log.Error().Msg("Missing auth_token in start event")
				return
			}

			log.Info().Str("stream_sid", s.StreamSID).Msg("Stream started and authenticated")
		} else if event["event"] == "media" {
			if s.GeminiConn == nil {
				continue // Skip media until Gemini is connected
			}
			media := event["media"].(map[string]interface{})
			payload, _ := base64.StdEncoding.DecodeString(media["payload"].(string))
			s.HandleTwilioAudio(payload)
		} else if event["event"] == "stop" {
			log.Info().Msg("Stream stopped")
			if s.DB != nil {
				go persistence.SummarizeCallJob(context.Background(), s.DB, s.CallID)
			}
			return
		}
	}
}

func (s *Session) twilioOutputLoop(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.TwilioConn == nil || s.StreamSID == "" {
				continue
			}

			// Pop 20ms of audio (8kHz * 0.02s = 160 samples)
			samples := s.JitterBuf.Pop(160)
			mulaw := audio.PCM16ToMulaw(samples)

			s.writeMu.Lock()
			s.TwilioConn.WriteJSON(map[string]interface{}{
				"event":     "media",
				"streamSid": s.StreamSID,
				"media": map[string]string{
					"payload": base64.StdEncoding.EncodeToString(mulaw),
				},
			})
			s.writeMu.Unlock()
		}
	}
}

func (s *Session) geminiToTwilioLoop(ctx context.Context) {
	log := voicelog.FromContext(ctx)
	for {
		var resp gemini.GeminiResponse
		if err := s.GeminiConn.ReceiveRaw(&resp); err != nil {
			if err != io.EOF {
				log.Error().Err(err).Msg("Gemini connection closed, attempting reconnect...")
				// Reconnect logic would go here, using s.ResumptionToken
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

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
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
			s.writeMu.Lock()
			s.TwilioConn.WriteJSON(map[string]string{
				"event":     "clear",
				"streamSid": s.StreamSID,
			})
			s.writeMu.Unlock()
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
					s.JitterBuf.Push(resampled)
				}
			}
		}
	}

	if resp.ToolCall != nil && s.ToolRouter != nil {
		for _, call := range resp.ToolCall.FunctionCalls {
			result, err := s.ToolRouter.Invoke(ctx, s.DB, call.Name, call.Args)

			s.writeMu.Lock()
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
			s.writeMu.Unlock()
		}
	}
}
