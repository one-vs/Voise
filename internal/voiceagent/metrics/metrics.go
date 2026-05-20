package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SessionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "voice_active_sessions_gauge",
		Help: "Number of currently active voice sessions",
	})

	SessionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "voice_session_duration_seconds",
		Help: "Duration of voice sessions in seconds",
	})

	FirstResponseLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "voice_first_response_latency_seconds",
		Help: "Latency of the first response from Gemini",
	})

	ToolInvocationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "tool_invocation_duration_seconds",
		Help: "Duration of tool invocations in seconds",
	}, []string{"name", "source"})

	ToolInvocationErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tool_invocation_errors_total",
		Help: "Total number of tool invocation errors",
	}, []string{"name", "source"})

	GeminiReconnects = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gemini_ws_reconnects_total",
		Help: "Total number of Gemini WebSocket reconnections",
	})

	AudioBufferUnderrun = promauto.NewCounter(prometheus.CounterOpts{
		Name: "audio_buffer_underrun_total",
		Help: "Total number of audio buffer underruns",
	})

	MCPCallErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "mcp_call_errors_total",
		Help: "Total number of MCP call errors",
	}, []string{"server"})
)
