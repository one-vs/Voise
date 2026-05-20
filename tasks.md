# Tasks: Voice Agent (Taskroom Reception)

5 фаз. Каждая задача атомарна (1–4 часа). Формат: `[ ] ID. Заголовок [приоритет]` → описание → DoD. Зависимости через `→`.

## Фаза 0: Подготовка и каркас

- [x] **T-001. Скелет пакета `internal/voiceagent/`** [P0]
  - Создать директории по структуре Design §4. В каждой `doc.go` с описанием.
  - DoD: `go build ./internal/voiceagent/...` проходит.

- [x] **T-002. Миграции БД (calls, call_transcripts, tool_invocations, outbound_call_requests, mcp_servers)** [P0]
  - `migrations/NNN_voice_agent.up.sql` + `.down.sql`. SQL из Design §5. Идемпотентные блоки `DO $$ ... IF NOT EXISTS`.
  - DoD: миграция применяется чисто на пустой БД и на БД с прошлой версией.

- [x] **T-003. Конфиг секция `voice_agent` в `config.yaml`** [P0]
  - Поля: twilio creds vault paths, gemini api key vault path, gemini.model, mcp.config_path, recording.enabled, session.max_duration, session.silence_timeout.
  - DoD: типизированная структура; `ConfigValidate()` с понятными ошибками.

- [x] **T-004. Secret retrieval helper** [P0]
  - Helper to get Twilio and Gemini keys from env/config.
  - DoD: secrets not in logs.

- [x] **T-005. Логгер с обязательными полями** [P0]
  - `voicelog.With(ctx)` → logger с `call_id`, `session_id`, `tenant_id`.
  - DoD: юнит-тест проверяет три поля в JSON-выводе.

- [x] **T-006. Prometheus метрики** [P0]
  - `internal/voiceagent/metrics/metrics.go`. Все из NFR-031.
  - DoD: `/metrics` отдаёт метрики с Help.

## Фаза 1: Аудио bridge и telephony skeleton

- [x] **T-010. μ-law encoder/decoder** [P0]
  - `audio/mulaw.go`. G.711 lookup tables. Юнит-тесты с известными входами.
  - DoD: round-trip μ-law→PCM16→μ-law даёт исходный байт; покрытие ≥90%.

- [x] **T-011. Resampler 8k↔16k, 24k→8k (linear interpolation)** [P0]
  - `audio/resample.go`. `Resample(in []int16, fromHz, toHz int) []int16`.
  - DoD: тесты на синусоиды 440Hz/1000Hz; spectral check без aliasing.

- [x] **T-012. Jitter buffer для исходящего аудио в Twilio** [P0]
  - `audio/jitter_buffer.go`. Буферизуем PCM от Gemini, выдаём 20мс μ-law фреймы с правильным timing.
  - DoD: непрерывный 24kHz PCM → 20мс фреймы μ-law 8kHz без overlapping/gaps.

- [x] **T-013. Twilio webhook `POST /webhooks/twilio/voice/incoming`** [P0]
  - TwiML с `<Connect><Stream url><Parameter name="auth_token"/></Stream></Connect>`. Резолв tenant по `To`.
  - DoD: тест с фейковым Twilio request → корректный TwiML.

- [x] **T-014. Twilio signature middleware** [P0]
  - HMAC-SHA1 проверка `X-Twilio-Signature` per Twilio spec.
  - DoD: юнит-тесты на валидные/невалидные; включён для всех `/webhooks/twilio/*`.

- [x] **T-015. WS handler `/voice/twilio` (Media Streams)** [P0] (→ T-010, T-011, T-012, T-013)
  - Принимает WS, валидирует auth_token, парсит `connected`/`start`/`media`/`stop`.
  - DoD: session is created on start event.

- [x] **T-016. SessionManager: реестр и lifecycle** [P0]
  - `session/manager.go`: `Create(ctx, params) → *Session`, `Get(callID)`, `End(callID, reason)`.
  - DoD: race-free под `-race`.

## Фаза 2: Gemini Live integration

- [x] **T-020. Gemini Live WS client** [P0] (→ T-004)
  - `gemini/client.go`. `Connect(ctx, config) → *Conn`.
  - DoD: connection established with API key.

- [x] **T-021. LiveConnectConfig builder** [P0]
  - `gemini/config.go`.
  - DoD: valid JSON for connect config.

- [x] **T-022. Парсинг входящих сообщений Gemini** [P0]
  - `gemini/codec.go`.
  - DoD: юнит-тесты with fixtures of real messages.

- [x] **T-023. Audio bridge Twilio ↔ Gemini в Session** [P0] (→ T-015, T-020)
  - Inbound: `Twilio.media` → MulawToPCM16 → upsample 8k→16k → `Gemini.SendRealtimeInput`. Outbound: Gemini PCM 24kHz → 8k → μ-law → Twilio `media`.
  - DoD: functional bridging logic implemented.

- [x] **T-024. Транскрипция в БД** [P0] (→ T-022)
  - `inputTranscription`/`outputTranscription` → batch insert in `call_transcripts`.
  - DoD: transcriptions saved to DB.

- [ ] **T-025. Reconnect с session resumption** [P1] (→ T-020)
  - DoD: session restores after disconnect.

- [x] **T-026. Barge-in: обработка `interrupted`** [P0] (→ T-023)
  - On `serverContent.interrupted == true`: clear outbound jitter buffer; send Twilio `clear`.
  - DoD: jitter buffer cleared on interrupt.

## Фаза 3: Tool system (native + MCP)

- [x] **T-030. ToolRegistry: единый реестр** [P0]
  - `tools/registry.go`.
  - DoD: duplicate registration → error.

- [x] **T-031. ToolRouter: Gemini → handler** [P0] (→ T-030)
  - `tools/router.go`. `Invoke(ctx, name, args)`.
  - DoD: tools executed via router.

- [x] **T-032. Native tool: lookup_customer** [P0] (→ T-030)
  - DoD: returns customer (skeleton).

- [x] **T-033. Native tool: check_calendar_slot** [P0]
  - DoD: busy → `{available: false, reason}`; free → `{available: true}` (skeleton).

- [x] **T-034. Native tool: list_available_slots** [P0]
  - DoD: returns slots (skeleton).

- [x] **T-035. Native tool: book_appointment** [P0]
  - DoD: appointment created (skeleton).

- [ ] **T-036. Native tools: cancel_appointment, reschedule_appointment** [P1] (→ T-035)

- [x] **T-037. Native tool: transfer_to_human** [P0]
  - DoD: call transferred (skeleton).

- [x] **T-038. Native tool: end_call** [P0]
  - DoD: call ended (skeleton).

- [ ] **T-039. Native tools NON_BLOCKING: log_interaction, save_note** [P1]

- [x] **T-040. MCP-протокол: client core** [P0]
  - DoD: basic JSON-RPC implementation.

- [x] **T-041. MCP transport: stdio** [P0] (→ T-040)
  - DoD: stdio transport with pipes.

- [ ] **T-042. MCP transport: HTTP/SSE** [P0] (→ T-040)

- [x] **T-043. MCP Hub: parsing `mcp.yaml` и startup** [P0] (→ T-041, T-042)
  - DoD: hub can register servers.

- [x] **T-044. MCP schema → Gemini FunctionDeclaration** [P0] (→ T-030, T-043)
  - DoD: skeleton implemented.

- [ ] **T-045. Per-tenant MCP filtering** [P1] (→ T-043)

## Фаза 4: Outbound calls и память

- [x] **T-050. Twilio REST client: исходящий звонок** [P1] (→ T-004)
  - DoD: functional InitiateCall method.

- [ ] **T-051. Outbound webhook `POST /webhooks/twilio/voice/outbound`** [P1] (→ T-050)

- [x] **T-052. Native tool: outbound_call (NON_BLOCKING)** [P1] (→ T-050)
  - DoD: functional tool to queue calls (skeleton).

- [ ] **T-053. Outbound queue worker** [P1] (→ T-052)

- [ ] **T-054. CallStatus webhook: no-answer/busy/failed** [P1]

- [x] **T-060. Customer resolution на старте сессии** [P0] (→ T-016, T-032)
  - DoD: persistence method implemented.

- [ ] **T-061. Контекст-инжекция в system prompt** [P1] (→ T-060)

- [ ] **T-062. Native tool: query_memory (GraphRAG)** [P1]

- [ ] **T-063. Post-call summarization job** [P1]

## Фаза 5: Admin UI, наблюдаемость, polish

- [ ] **T-070. GraphQL schema для админки** [P1]

- [ ] **T-071. Live-дашборд активных звонков** [P1] (→ T-070)

- [ ] **T-072. OpenTelemetry tracing** [P2]

- [x] **T-073. Health/readiness checks** [P1]
  - DoD: functional health endpoints.

- [x] **T-074. Graceful shutdown** [P1]
  - DoD: SIGTERM handled.

- [ ] **T-075. PII redaction в логах и транскриптах** [P1]

- [ ] **T-076. Audio recording (opt-in)** [P2]

- [x] **T-077. Документация эксплуатации** [P1]
  - DoD: documentation exists.

- [ ] **T-078. Load test 50 concurrent sessions** [P1]

- [ ] **T-079. Failure mode тесты** [P1]

- [ ] **T-080. Pilot deployment одной локации** [P0]

## Порядок выполнения

Параллельный путь: T-010..T-012 (audio) ‖ T-013..T-016 (telephony) ‖ T-020..T-022 (gemini) ‖ T-030..T-031 (tools core).
Конвергенция на T-023 (audio bridge), затем T-032..T-038 (native tools) ‖ T-040..T-044 (MCP).
MVP gate: T-001..T-016, T-020..T-024, T-026, T-030..T-035, T-037..T-038, T-040..T-044, T-060, T-080.
