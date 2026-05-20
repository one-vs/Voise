package session

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"voise/internal/voiceagent/audio"
	"voise/internal/voiceagent/gemini"
	"voise/internal/voiceagent/voicelog"
)

type Session struct {
	ID         uuid.UUID
	CallID     uuid.UUID
	TwilioConn *websocket.Conn
	GeminiConn *gemini.Conn

	JitterBuf  *audio.JitterBuffer

	mu sync.Mutex
}

func (s *Session) Run(ctx context.Context) {
	log := voicelog.FromContext(ctx)
	log.Info().Msg("Session loop started")

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

		if event["event"] == "media" {
			media := event["media"].(map[string]interface{})
			payload, _ := base64.StdEncoding.DecodeString(media["payload"].(string))
			s.HandleTwilioAudio(payload)
		} else if event["event"] == "stop" {
			return
		}
	}
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
		s.HandleGeminiResponse(&resp)
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

func (s *Session) HandleGeminiResponse(resp *gemini.GeminiResponse) {
	if resp.ServerContent != nil {
		if resp.ServerContent.Interrupted {
			s.JitterBuf.Clear()
			s.TwilioConn.WriteJSON(map[string]string{"event": "clear"})
		}

		if resp.ServerContent.ModelTurn != nil {
			for _, part := range resp.ServerContent.ModelTurn.Parts {
				if part.InlineData != nil {
					audioBytes, _ := base64.StdEncoding.DecodeString(part.InlineData.Data)
					pcm16 := audio.DecodePCM16LE(audioBytes)
					resampled := audio.Resample(pcm16, 24000, 8000)
					mulaw := audio.PCM16ToMulaw(resampled)

					s.TwilioConn.WriteJSON(map[string]interface{}{
						"event": "media",
						"media": map[string]string{
							"payload": base64.StdEncoding.EncodeToString(mulaw),
						},
					})
				}
			}
		}
	}
}
