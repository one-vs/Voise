-- 001_voice_agent.up.sql

CREATE TABLE IF NOT EXISTS calls (
  id                  UUID PRIMARY KEY,
  tenant_id           UUID NOT NULL,
  location_id         UUID NOT NULL,
  twilio_call_sid     TEXT UNIQUE NOT NULL,
  direction           TEXT NOT NULL CHECK (direction IN ('inbound','outbound')),
  from_number         TEXT NOT NULL,
  to_number           TEXT NOT NULL,
  customer_id         UUID,
  status              TEXT NOT NULL,
  started_at          TIMESTAMPTZ NOT NULL,
  answered_at         TIMESTAMPTZ,
  ended_at            TIMESTAMPTZ,
  duration_seconds    INT,
  recording_url       TEXT,
  outcome             TEXT,
  agent_summary       TEXT,
  metadata            JSONB NOT NULL DEFAULT '{}',
  created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_calls_tenant_started ON calls(tenant_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_calls_customer ON calls(customer_id) WHERE customer_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS call_transcripts (
  id              UUID PRIMARY KEY,
  call_id         UUID NOT NULL REFERENCES calls(id) ON DELETE CASCADE,
  speaker         TEXT NOT NULL CHECK (speaker IN ('user','agent','system')),
  content         TEXT NOT NULL,
  ts_offset_ms    INT NOT NULL,
  is_final        BOOL NOT NULL DEFAULT true,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_transcripts_call ON call_transcripts(call_id, ts_offset_ms);

CREATE TABLE IF NOT EXISTS tool_invocations (
  id              UUID PRIMARY KEY,
  call_id         UUID NOT NULL REFERENCES calls(id) ON DELETE CASCADE,
  tool_name       TEXT NOT NULL,
  tool_source     TEXT NOT NULL,
  arguments       JSONB NOT NULL,
  result          JSONB,
  error           TEXT,
  duration_ms     INT,
  ts_offset_ms    INT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_tools_call ON tool_invocations(call_id, ts_offset_ms);

CREATE TABLE IF NOT EXISTS outbound_call_requests (
  id              UUID PRIMARY KEY,
  tenant_id       UUID NOT NULL,
  to_number       TEXT NOT NULL,
  reason          TEXT NOT NULL,
  context         JSONB NOT NULL,
  scheduled_at    TIMESTAMPTZ NOT NULL,
  status          TEXT NOT NULL,
  call_id         UUID REFERENCES calls(id),
  attempts        INT NOT NULL DEFAULT 0,
  last_error      TEXT,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS mcp_servers (
  id              UUID PRIMARY KEY,
  tenant_id       UUID,  -- NULL = глобальный
  name            TEXT NOT NULL,
  transport       TEXT NOT NULL CHECK (transport IN ('stdio','http','sse')),
  config          JSONB NOT NULL,
  enabled         BOOL NOT NULL DEFAULT true,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tenant_id, name)
);
