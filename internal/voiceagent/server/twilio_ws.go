package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"voise/internal/voiceagent/gemini"
	"voise/internal/voiceagent/session"
	"voise/internal/voiceagent/voicelog"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type TwilioEvent struct {
	Event     string                 `json:"event"`
	StreamSID string                 `json:"streamSid"`
	Media     *TwilioMedia           `json:"media,omitempty"`
	Start     *TwilioStart           `json:"start,omitempty"`
}

type TwilioMedia struct {
	Payload string `json:"payload"`
}

type TwilioStart struct {
	StreamSID string `json:"streamSid"`
	CallSID   string `json:"callSid"`
	StartData map[string]interface{} `json:"startData"`
}

func HandleTwilioWS(mgr *session.Manager, geminiClient *gemini.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simple token validation
		token := r.URL.Query().Get("auth_token")
		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			voicelog.Logger.Error().Err(err).Msg("Failed to upgrade Twilio WS")
			return
		}
		defer conn.Close()

		var s *session.Session

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var event TwilioEvent
			if err := json.Unmarshal(message, &event); err != nil {
				continue
			}

			switch event.Event {
			case "start":
				voicelog.Logger.Info().Str("stream_sid", event.Start.StreamSID).Msg("Twilio Stream started")
				// In a real app, we'd resolve call details. For now, create a session.
				s = mgr.Create(r.Context(), uuid.New())
				s.TwilioConn = conn
				s.StreamSID = event.Start.StreamSID
				s.GeminiClient = geminiClient

				// Try to lookup customer by From number
				// For now, we don't have the From number in this simple WS event,
				// it's usually passed in the webhook call. Let's assume it's in metadata or handled at session level.

				go s.Run(r.Context())
			case "media":
				if s != nil {
					payload, _ := base64.StdEncoding.DecodeString(event.Media.Payload)
					s.HandleTwilioAudio(payload)
				}
			case "stop":
				voicelog.Logger.Info().Msg("Twilio Stream stopped")
				if s != nil {
					mgr.Remove(s.ID)
				}
				return
			}
		}
	}
}
