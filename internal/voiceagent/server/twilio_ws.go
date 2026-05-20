package server

import (
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
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			voicelog.Logger.Error().Err(err).Msg("Failed to upgrade Twilio WS")
			return
		}
		defer conn.Close()

		s := mgr.Create(r.Context(), uuid.New())
		s.TwilioConn = conn
		s.GeminiClient = geminiClient

		defer mgr.Remove(s.ID)
		s.Run(r.Context())
	}
}
