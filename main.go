package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"voise/internal/voiceagent/config"
	"voise/internal/voiceagent/server"
	"voise/internal/voiceagent/session"
	"voise/internal/voiceagent/tools"
	"voise/internal/voiceagent/tools/native"
	"voise/internal/voiceagent/voicelog"
)

func main() {
	// Initialize logging
	log := voicelog.Logger

	// Initialize configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load config, using defaults")
		cfg = &config.Config{}
	}

	// Initialize registry and register native tools
	registry := tools.NewToolRegistry()
	registry.Register("lookup_customer", native.LookupCustomer)
	registry.Register("transfer_to_human", native.TransferToHuman)
	registry.Register("end_call", native.EndCall)

	// Initialize managers
	sessionMgr := session.NewManager()

	mux := http.NewServeMux()

	// Twilio Webhooks
	mux.HandleFunc("/webhooks/twilio/voice/incoming", server.HandleIncomingCall("/voice/twilio/ws"))
	mux.HandleFunc("/voice/twilio/ws", server.HandleTwilioWS(sessionMgr))

	// Health checks
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	log.Info().Msg("Server started on :8080")

	// Use cfg to avoid unused warning
	_ = cfg

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exiting")
}
