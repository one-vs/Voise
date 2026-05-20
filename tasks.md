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

- [ ] **T-004. Vault-интеграция для Twilio и Gemini** [P0]
  - Хелперы `GetTwilioCreds(ctx)`, `GetGeminiKey(ctx)`.
  - DoD: секреты не попадают в логи (проверить через структурированный логгер).

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
  - TwiML с `<Connect><Stream url><Parameter name="auth_token"/></Stream></Connect>`. Резолв tenant по `To`. Одноразовый auth_token TTL 60с в Redis.
  - DoD: тест с фейковым Twilio request → корректный TwiML; невалидная подпись → 403.

- [x] **T-014. Twilio signature middleware** [P0]
  - HMAC-SHA1 проверка `X-Twilio-Signature` per Twilio spec.
  - DoD: юнит-тесты на валидные/невалидные; включён для всех `/webhooks/twilio/*`.

- [x] **T-015. WS handler `/voice/twilio` (Media Streams)** [P0] (→ T-010, T-011, T-012, T-013)
  - Принимает WS, валидирует auth_token, парсит `connected`/`start`/`media`/`stop`. На `start` — `Session` через `SessionManager`. На `media` — base64 → μ-law → PCM16 16k → очередь в Gemini.
  - DoD: тест с замоканным Twilio WS-клиентом → backend декодирует и складывает в очередь.

- [x] **T-016. SessionManager: реестр и lifecycle** [P0]
  - `session/manager.go`: `Create(ctx, params) → *Session`, `Get(callID)`, `End(callID, reason)`. Per-session goroutine с select по каналам.
  - DoD: race-free под `-race`; смерть сессии освобождает все ресурсы.

## Фаза 2: Gemini Live integration

- [x] **T-020. Gemini Live WS client** [P0] (→ T-004)
  - `gemini/client.go`. `Connect(ctx, config) → *Conn`. Сообщения: `setup`, `clientContent`, `realtimeInput`, `toolResponse`.
  - DoD: тест с моком Gemini WS-сервера: setup → audio → audio chunks обратно.

- [x] **T-021. LiveConnectConfig builder** [P0]
  - `gemini/config.go`. Собирает из tenant settings: responseModalities, transcription, voice, systemInstruction, tools.
  - DoD: для тестового tenant — валидный JSON, smoke против Gemini API в CI.

- [x] **T-022. Парсинг входящих сообщений Gemini** [P0]
  - `gemini/codec.go`. Типы: `ServerContent`, `ToolCall`, `ToolCallCancellation`, `SetupComplete`, `GoAway`, `SessionResumptionUpdate`. Multi-part события.
  - DoD: юнит-тесты with fixtures of real messages.

- [x] **T-023. Audio bridge Twilio ↔ Gemini в Session** [P0] (→ T-015, T-020)
  - Inbound: `Twilio.media` → MulawToPCM16 → upsample 8k→16k → `Gemini.SendRealtimeInput`. Outbound: Gemini PCM 24kHz → 8k → μ-law → Twilio `media`.
  - DoD: e2e smoke with real Twilio number → voice is heard; barge-in works.

- [ ] **T-024. Транскрипция в БД** [P0] (→ T-022)
  - `inputTranscription`/`outputTranscription` → buffer ≤500ms or ≤5 fragments → batch insert in `call_transcripts`.
  - DoD: after call, DB has transcript of both sides with correct offset.

- [ ] **T-025. Reconnect с session resumption** [P1] (→ T-020)
  - On `GoAway`/WS disconnect: backoff 200/500/1000ms, reconnect with `sessionResumptionHandle`.
  - DoD: chaos-test with forced closure → session restores, pause <3s.

- [x] **T-026. Barge-in: обработка `interrupted`** [P0] (→ T-023)
  - On `serverContent.interrupted == true`: clear outbound jitter buffer; send Twilio `clear`; metric `voice_interruptions_total`.
  - DoD: test: simulate interrupt → outbound queue empties in <100ms.

## Фаза 3: Tool system (native + MCP)

- [x] **T-030. ToolRegistry: единый реестр** [P0]
  - `tools/registry.go`. `Register`, `Get`, `ListForGemini`. Namespace for MCP: `{source}.{tool}`.
  - DoD: duplicate registration → error; listing forms correct JSON.

- [x] **T-031. ToolRouter: Gemini → handler** [P0] (→ T-030)
  - `tools/router.go`. `Invoke(ctx, name, args, behavior) → (result, error)`. Log in `tool_invocations`. Timeout 8s, retry 1 time for idempotent.
  - DoD: native call returns result; timeout → structured error.

- [x] **T-032. Native tool: lookup_customer** [P0] (→ T-030)
  - Schema: `{phone?, name?}`. Filter by tenant_id.
  - DoD: returns customer by phone; null if not found; doesn't see other tenant.

- [ ] **T-033. Native tool: check_calendar_slot** [P0]
  - Schema: `{service_id, master_id?, datetime}`. Check in `appointments` + Google Calendar if connected.
  - DoD: busy → `{available: false, reason}`; free → `{available: true}`.

- [ ] **T-034. Native tool: list_available_slots** [P0]
  - Schema: `{service_id, master_id?, date_from, date_to, count?}` → `[{datetime, master_id}]`.
  - DoD: test on test data considering working hours and existing appointments.

- [ ] **T-035. Native tool: book_appointment** [P0]
  - Schema: `{customer_id, service_id, master_id, datetime}`. Creates appointment, SMS-confirmation async. Idempotency: repeat in 60s → same appointment.
  - DoD: appointment created; repeat — same id; conflicting slot — error.

- [ ] **T-036. Native tools: cancel_appointment, reschedule_appointment** [P1] (→ T-035)
  - DoD: each has happy/error unit test.

- [x] **T-037. Native tool: transfer_to_human** [P0]
  - Schema: `{reason, target_number?}`. Closes Gemini gracefully, TwiML on `<Dial>`.
  - DoD: e2e: call → client on target_number.

- [x] **T-038. Native tool: end_call** [P0]
  - Correctly ends Gemini, sends Twilio Hangup.
  - DoD: after call session.state == ended, WS closed.

- [ ] **T-039. Native tools NON_BLOCKING: log_interaction, save_note** [P1]
  - Return `{queued: true}` immediately, executed via worker. Final `FunctionResponse(scheduling: WHEN_IDLE)`.
  - DoD: model receives immediate response, in ≤2s — final without blocking speech.

- [x] **T-040. MCP-протокол: client core** [P0]
  - `mcp/client.go`. Abstract `Transport`. Message types per MCP spec. initialize, tools/list, tools/call.
  - DoD: unit tests with mocks: handshake → list → call → result serialized/parsed.

- [x] **T-041. MCP transport: stdio** [P0] (→ T-040)
  - `mcp/stdio_transport.go`. `exec.Command` with pipes, JSON-RPC over stdin/stdout, stderr → log. Reconnect on crash (max 3 attempts with backoff).
  - DoD: integration test with real `@modelcontextprotocol/server-filesystem` if available in CI.

- [ ] **T-042. MCP transport: HTTP/SSE** [P0] (→ T-040)
  - `mcp/http_transport.go`. HTTP POST for JSON-RPC, SSE for streaming, Bearer/OAuth auth.
  - DoD: test with mock HTTP server: request → SSE response → parsing.

- [x] **T-043. MCP Hub: parsing `mcp.yaml` и startup** [P0] (→ T-041, T-042)
  - `mcp/hub.go`. At startup: YAML → lift enabled servers → registration in `ToolRegistry`. One crash — others work.
  - DoD: test: 3 servers, one fails at initialize → 2 working available.

- [x] **T-044. MCP schema → Gemini FunctionDeclaration** [P0] (→ T-030, T-043)
  - `tools/declarations.go`. JSON Schema draft-07 → Gemini schema (subset OpenAPI). `oneOf`/`anyOf` (Gemini doesn't support) → flat or drop with warning.
  - DoD: юнит-тесты; smoke: connect google-calendar → Gemini accepts declarations.

- [ ] **T-045. Per-tenant MCP filtering** [P1] (→ T-043)
  - When creating session: filter MCP tools by `mcp_servers.tenant_id`.
  - DoD: tenant A sees its + global; doesn't see others.

## Фаза 4: Outbound calls и память

- [ ] **T-050. Twilio REST client: исходящий звонок** [P1] (→ T-004)
  - `outbound/initiator.go`. `InitiateCall(ctx, to, from, twiml_url) → call_sid`.
  - DoD: integration test in Twilio test mode returns call_sid.

- [ ] **T-051. Outbound webhook `POST /webhooks/twilio/voice/outbound`** [P1] (→ T-050)
  - On pickup by subscriber Twilio calls this webhook → TwiML with `<Connect><Stream>` and session_id.
  - DoD: e2e: outbound → agent answers on pickup with its script.

- [ ] **T-052. Native tool: outbound_call (NON_BLOCKING)** [P1] (→ T-050)
  - Schema: `{to, reason, context, schedule_at?}`. Entry in `outbound_call_requests`. `schedule_at` in past or absent — immediate, else — in queue.
  - DoD: test: tool creates entry + triggers Twilio call; FunctionResponse returns status.

- [ ] **T-053. Outbound queue worker** [P1] (→ T-052)
  - Polls `outbound_call_requests` status=queued + scheduled_at ≤ now. Rate limit per tenant (default 5 concurrent).
  - DoD: test: 10 requests with limit 5 → 5 in work, 5 waiting; after completion — next.

- [ ] **T-054. CallStatus webhook: no-answer/busy/failed** [P1]
  - `CallStatus=no-answer` → `outbound_call_requests.status=failed`, optional retry with backoff.
  - DoD: test: simulate no-answer → entry failed.

- [ ] **T-060. Customer resolution на старте сессии** [P0] (→ T-016, T-032)
  - In `Session.start`: `lookup_customer(phone=From)` → `SessionState.CustomerID`.
  - DoD: test: inbound from known number → customer_id filled; unknown → null.

- [ ] **T-061. Контекст-инжекция в system prompt** [P1] (→ T-060)
  - If customer found: 3 last `agent_summary` + last `appointments` (≤5) → render to template.
  - DoD: test: first Gemini message for known customer contains their name.

- [ ] **T-062. Native tool: query_memory (GraphRAG)** [P1]
  - Reuse Taskroom engine: embedding query → pgvector top-K → format.
  - DoD: test: fact saved via `save_note` → query_memory finds it.

- [ ] **T-063. Post-call summarization job** [P1]
  - On `ended_at` enqueue: transcript → Gemini text API → `agent_summary`, `outcome`, `sentiment`.
  - DoD: test: after call fields in DB are filled; job is idempotent.

## Фаза 5: Admin UI, наблюдаемость, polish

- [ ] **T-070. GraphQL schema для админки** [P1]
  - Queries: `calls(filter, pagination)`, `call(id)`, `activeCalls(locationId)`. Subscription: `callTranscriptUpdated(callId)`. Mutation: `endCall(id, reason)`.
  - DoD: schema is valid, resolver covered by unit tests.

- [ ] **T-071. Live-дашборд активных звонков** [P1] (→ T-070)
  - Subscription `activeCallsUpdated` → list of active, duration, last transcript fragment.
  - DoD: integration test: new call appears in subscription in realtime.

- [ ] **T-072. OpenTelemetry tracing** [P2]
  - Spans: session, tool calls, MCP calls, Gemini round-trips.
  - DoD: trace on test call visible in Jaeger/Tempo.

- [x] **T-073. Health/readiness checks** [P1]
  - `/healthz` (process alive), `/readyz` (DB + Gemini reachable + at least one MCP healthy).
  - DoD: with DB off readyz → 503.

- [x] **T-074. Graceful shutdown** [P1]
  - SIGTERM: new calls rejected with TwiML alt message; active calls finish or 5 minutes.
  - DoD: e2e: SIGTERM during call → client finishes listening.

- [ ] **T-075. PII redaction в логах и транскриптах** [P1]
  - Regex for card numbers (Luhn), SSN, passports. Optional per-tenant.
  - DoD: text with test card → in logs mask `**** **** **** 1234`.

- [ ] **T-076. Audio recording (opt-in)** [P2]
  - Twilio recording → S3, link in `calls.recording_url`. Start only after consent at beginning.
  - DoD: test: with consent — recording_url filled, without — null.

- [x] **T-077. Документация эксплуатации** [P1]
  - README: add MCP, add location, add native tool. Runbook: typical incidents.
  - DoD: external developer adds test MCP without help following README.

- [ ] **T-078. Load test 50 concurrent sessions** [P1]
  - Script: simulates 50 parallel calls via Twilio test or mocks.
  - DoD: NFR-001 (latency p95 ≤1500ms) and NFR-003 (50 sessions without degradation) met.

- [ ] **T-079. Failure mode тесты** [P1]
  - Chaos-tests: Gemini WS drop, MCP crash, Twilio WS drop, audio buffer underrun, silence timeout.
  - DoD: for each scenario behavior matches Message §10.

- [ ] **T-080. Pilot deployment одной локации** [P0]
  - Deploy staging, register Twilio number, real test calls with team.
  - DoD: 10 successful test calls with different scenarios (booking, info, transfer).

## Порядок выполнения

Параллельный путь: T-010..T-012 (audio) ‖ T-013..T-016 (telephony) ‖ T-020..T-022 (gemini) ‖ T-030..T-031 (tools core).
Конвергенция на T-023 (audio bridge), затем T-032..T-038 (native tools) ‖ T-040..T-044 (MCP).
MVP gate: T-001..T-016, T-020..T-024, T-026, T-030..T-035, T-037..T-038, T-040..T-044, T-060, T-080.
