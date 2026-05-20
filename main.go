package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"voise/internal/voiceagent/config"
	"voise/internal/voiceagent/gemini"
	"voise/internal/voiceagent/mcp"
	"voise/internal/voiceagent/server"
	"voise/internal/voiceagent/session"
	"voise/internal/voiceagent/tools"
	"voise/internal/voiceagent/tools/native"
	"voise/internal/voiceagent/voicelog"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Initialize logging
	log := voicelog.Logger

	// Initialize Tracer
	tp, _ := voicelog.InitTracer()
	if tp != nil {
		defer tp.Shutdown(context.Background())
	}

	// Initialize configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load config, using defaults")
		cfg = &config.Config{}
	}

	// Initialize Database
	var db *sqlx.DB
	if cfg.VoiceAgent.DatabaseURL != "" {
		db, err = sqlx.Connect("postgres", cfg.VoiceAgent.DatabaseURL)
		if err != nil {
			log.Error().Err(err).Msg("Failed to connect to database")
		} else {
			log.Info().Msg("Connected to database")
			defer db.Close()
		}
	}

	// Initialize registry and register native tools
	registry := tools.NewToolRegistry()
	registry.Register("lookup_customer", native.LookupCustomer)
	registry.Register("transfer_to_human", native.TransferToHuman)
	registry.Register("end_call", native.EndCall)
	registry.Register("check_calendar_slot", native.CheckCalendarSlot)
	registry.Register("list_available_slots", native.ListAvailableSlots)
	registry.Register("book_appointment", native.BookAppointment)
	registry.Register("cancel_appointment", native.CancelAppointment)
	registry.Register("reschedule_appointment", native.RescheduleAppointment)
	registry.Register("log_interaction", native.LogInteraction)
	registry.Register("save_note", native.SaveNote)
	registry.Register("query_memory", native.QueryMemory)
	registry.Register("outbound_call", native.OutboundCall)
	router := tools.NewToolRouter(registry)

	// Initialize MCP Hub
	mcpHub := mcp.NewHub()
	if cfg.VoiceAgent.MCP.ConfigPath != "" {
		if err := mcpHub.LoadConfig(cfg.VoiceAgent.MCP.ConfigPath); err != nil {
			log.Warn().Err(err).Msg("Failed to load MCP config")
		}
	}

	// Initialize managers
	sessionMgr := session.NewManager(db, router, mcpHub)
	geminiClient := gemini.NewClient(cfg.VoiceAgent.Gemini.APIKey, cfg.VoiceAgent.Gemini.Model)

	mux := http.NewServeMux()

	// Twilio Webhooks
	incomingHandler := server.HandleIncomingCall("/voice/twilio/ws")
	if cfg.VoiceAgent.Twilio.AuthToken != "" {
		incomingHandler = server.TwilioSignatureMiddleware(cfg.VoiceAgent.Twilio.AuthToken, incomingHandler)
	}
	mux.HandleFunc("/webhooks/twilio/voice/incoming", incomingHandler)
	mux.HandleFunc("/voice/twilio/ws", server.HandleTwilioWS(sessionMgr, geminiClient))

	// Health checks
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// GraphQL API
	mux.HandleFunc("/graphql", server.HandleGraphQL(db))

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
