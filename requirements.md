# Requirements: Voice Agent (Taskroom Reception)

Формат: ID, тип (FR/NFR), приоритет (P0/P1/P2), проверяемый критерий приёмки.

## 1. Функциональные требования

### 1.1. Телефония и звонки

**FR-001 (P0): Приём входящего звонка.** Twilio-номер локации, агент поднимает трубку с приветствием за ≤2с.
- AC: TwiML возвращается на webhook за ≤500мс; WS Twilio↔backend устанавливается за ≤500мс; первое аудио от Gemini в звонок за ≤1500мс.

**FR-002 (P0): Bidirectional аудио через Twilio Media Streams.**
- AC: Twilio шлёт inbound μ-law 8kHz; backend конвертирует в PCM 16kHz и шлёт в Gemini; ответ PCM 24kHz конвертируется обратно в μ-law 8kHz без слышимых артефактов.

**FR-003 (P0): Native real-time через Gemini Live API.**
- AC: используется модель из семейства `gemini-*-live-*`; конфиг включает `responseModalities: [AUDIO]`, `inputTranscription`, `outputTranscription`.

**FR-004 (P0): Barge-in / прерывания.**
- AC: при `interrupted` от Gemini — outbound буфер очищается, `clear` event в Twilio за ≤100мс; новое аудио без артефактов наложения.

**FR-005 (P0): Завершение звонка.**
- AC: на `stop` от Twilio сессия Gemini закрывается за ≤500мс, запись в `calls.ended_at`, фоновое суммирование запущено.

**FR-006 (P1): Исходящие звонки, инициированные агентом.**
- AC: вызов `outbound_call(to, reason, context)` создаёт запись в `outbound_call_requests`, делает Twilio REST call, при pickup открывается WS-сессия с правильным system prompt.

**FR-007 (P1): Перевод на оператора.**
- AC: `transfer_to_human(reason, target_number)` корректно завершает Gemini, обновляет TwiML на `<Dial>`, клиент попадает на живого человека.

**FR-008 (P2): Запись звонков с согласием.**
- AC: Twilio пишет в S3, ссылка в `calls.recording_url`; запись стартует только после фразы согласия + положительного ответа клиента.

### 1.2. Инструменты (function calling)

**FR-010 (P0): Native tools — все 15 из Design §8.** Корректные JSON-schemas, валидация входа, лог в `tool_invocations`.
- AC: для каждого юнит-тест (happy + минимум 1 error); вызов из Gemini маршрутизируется и возвращает результат в SLO behavior (BLOCKING <500мс / NON_BLOCKING async).

**FR-011 (P0): MCP Hub — динамическая регистрация.** stdio и HTTP/SSE.
- AC: при старте читается `mcp.yaml`, для каждого enabled делается `initialize` + `tools/list`, инструменты регистрируются в `ToolRegistry`, конвертируются в `FunctionDeclaration`.

**FR-012 (P0): Добавление нового MCP без правки Go-кода.**
- AC: новая запись в `mcp.yaml` + рестарт → новый MCP подключён. Тест: добавить `@modelcontextprotocol/server-filesystem`, агент может вызвать `read_file`.

**FR-013 (P0): Tool routing.** Единый `ToolRouter.Invoke(name, args)` различает native vs MCP.
- AC: native не уходит наружу процесса; MCP идёт через правильный transport; коллизии резолвятся через namespace `{mcp_server}.{tool}`.

**FR-014 (P1): NON_BLOCKING инструменты.**
- AC: для tools с `behavior: NON_BLOCKING` модель продолжает говорить пока tool выполняется; результат через `FunctionResponse` с `scheduling: INTERRUPT` или `WHEN_IDLE`.

**FR-015 (P1): Multi-tenant tool config.**
- AC: в `mcp_servers` поле `tenant_id` фильтрует; локация А не видит MCP локации Б; глобальные (tenant_id=NULL) доступны всем.

**FR-016 (P2): Tool timeout и retry.**
- AC: tool call >8с возвращает structured error; для идемпотентных tools — 1 retry с backoff 500мс.

### 1.3. Память и контекст

**FR-020 (P0): Идентификация клиента по caller ID.**
- AC: при входящем `From` ищется в `customers`, найденный клиент инжектится в system prompt.

**FR-021 (P1): Загрузка истории.**
- AC: для known customer подгружаются 3–5 последних `agent_summary` + последние `appointments`, инжектятся в prompt.

**FR-022 (P1): GraphRAG-запросы во время звонка.**
- AC: `query_memory(query)` возвращает релевантные фрагменты через pgvector similarity.

**FR-023 (P1): Пост-звонковое суммирование.**
- AC: в течение 30с после `ended_at` фоновый воркер генерит `agent_summary`, обновляет память клиента, апдейтит CRM.

### 1.4. Транскрипция и логирование

**FR-030 (P0): Real-time транскрипция обеих сторон.**
- AC: `inputTranscription` и `outputTranscription` от Gemini пишутся в `call_transcripts` с offset; UI Taskroom показывает в реальном времени через GraphQL subscription.

**FR-031 (P0): Полный аудит tool calls.**
- AC: каждый вызов в `tool_invocations` с arguments, result, duration_ms; чувствительные поля редактируются.

**FR-032 (P2): Sentiment и категоризация.**
- AC: фоновый джоб проставляет `outcome` (booked/cancelled/info_provided/transferred/failed) и `sentiment` (positive/neutral/negative).

### 1.5. Мультитенантность

**FR-040 (P0): Резолв tenant по Twilio-номеру.**
- AC: webhook по `To` находит `locations.id` и `tenants.id`, все операции идут с этим контекстом.

**FR-041 (P0): Изоляция данных.**
- AC: SQL фильтруется по `tenant_id` на уровне репозиториев; запрос данных tenant A с контекстом tenant B возвращает пусто / 403.

**FR-042 (P1): Per-tenant system prompts.**
- AC: локация может переопределить шаблон в `location_settings.prompt_overrides`; рендеринг через text/template.

### 1.6. Управление и наблюдаемость

**FR-050 (P1): Admin GraphQL API.**
- AC: queries `calls`, `call(id)`, `activeCalls(locationId)`; subscription `callTranscriptUpdated(callId)`; mutation `endCall(id, reason)`.

**FR-051 (P2): Управление MCP через UI.**
- AC: список MCP-серверов, кнопка enable/disable, просмотр schemas инструментов.

## 2. Нефункциональные требования

### 2.1. Производительность и latency

**NFR-001 (P0): End-to-end latency реплики.** Медиана ≤800мс, p95 ≤1500мс (VAD-end Gemini → первый PCM в Twilio).

**NFR-002 (P0): Tool call latency.** BLOCKING native p95 ≤300мс. BLOCKING MCP p95 ≤1500мс.

**NFR-003 (P1): Concurrent sessions.** Одна инстанция (4 vCPU / 8GB RAM) держит ≥50 одновременных звонков без деградации.

**NFR-004 (P1): Audio quality.** Resampling без перцептивных артефактов (PESQ ≥3.5 на тестовом корпусе); buffer underrun rate <0.1% от длительности звонка.

### 2.2. Надёжность

**NFR-010 (P0): Reconnection.** При WS-разрыве с Gemini — auto-reconnect с `session_resumption_handle`, прозрачно в окне ≤3с.

**NFR-011 (P0): No data loss.** Транскрипты и tool calls пишутся в БД с батчингом ≤500мс; при крэше процесса в БД остаётся всё до момента крэша.

**NFR-012 (P1): Graceful shutdown.** SIGTERM → новые звонки отклоняются, активные дозваниваются до конца или 5 минут.

**NFR-013 (P1): Падение MCP не валит сессию.** Инструменты упавшего MCP помечаются недоступными, остальные работают; модели — structured error.

### 2.3. Безопасность

**NFR-020 (P0): Twilio webhook signature.** HMAC-SHA1 валидация `X-Twilio-Signature`; невалидная → 403.

**NFR-021 (P0): WSS auth.** WS от Twilio принимается только с валидным одноразовым токеном из TwiML параметра; TTL 60с.

**NFR-022 (P0): Secrets в Vault.** Все API-ключи в HashiCorp Vault; не попадают в логи, не коммитятся.

**NFR-023 (P1): PII redaction в логах.** Номера карт, SSN, паспорта маскируются regex перед записью в лог и транскрипт (опционально per-tenant).

**NFR-024 (P1): TLS everywhere.** Все эндпоинты HTTPS/WSS; mTLS для MCP HTTP где возможно.

### 2.4. Наблюдаемость

**NFR-030 (P1): Структурированные логи.** JSON с обязательными `call_id`, `session_id`, `tenant_id`, `level`, `ts`, `event`.

**NFR-031 (P1): Prometheus метрики.** Минимум: `voice_session_duration_seconds`, `voice_active_sessions_gauge`, `voice_first_response_latency_seconds`, `tool_invocation_duration_seconds{name,source}`, `tool_invocation_errors_total{name,source}`, `gemini_ws_reconnects_total`, `audio_buffer_underrun_total`, `mcp_call_errors_total{server}`.

**NFR-032 (P2): Distributed tracing.** OpenTelemetry: spans для сессии, tool calls, MCP-вызовов, Gemini round-trips.

### 2.5. Совместимость

**NFR-040 (P0): MCP spec.** Соответствие MCP-спецификации (initialize, tools/list, tools/call); версия протокола указывается явно.

**NFR-041 (P0): Twilio Media Streams spec.** Поддержка событий: `connected`, `start`, `media`, `dtmf`, `mark`, `stop`; outbound: `media`, `mark`, `clear`.

**NFR-042 (P1): Multilingual.** Минимум RU, EN, ES; язык per-location.

### 2.6. Эксплуатация

**NFR-050 (P1): Конфиг через файлы + env.** Все настройки через `config.yaml` + env vars; перезагрузка SIGHUP где возможно.

**NFR-051 (P1): Health/readiness checks.** `/healthz` (процесс жив) и `/readyz` (БД + Gemini reachable + хотя бы один MCP healthy).

**NFR-052 (P2): Документация эксплуатации.** README по добавлению MCP, локации, native tools; runbook для типовых инцидентов.

## 3. Ограничения и допущения

- К0: используется инфраструктура Taskroom (Postgres, Vault, GraphQL gateway, Twilio account на финской компании).
- К1: Gemini Live API доступен по API-ключу с квотами под мультитенантную нагрузку.
- К2: на первой итерации только RU и EN; остальные языки P2.
- К3: запись звонков opt-in per tenant (юридически безопасный дефолт OFF).
- К4: агент не делает финансовых транзакций — это P2/P3, после compliance.

## 4. Критерии приёмки фазы 1 (MVP)

Минимально достаточный набор для пилота с одной локацией:
- FR-001..005 (входящий звонок, аудио, native Gemini, barge-in, завершение)
- FR-010, 011, 012, 013 (native tools + MCP hub + динамическая регистрация)
- FR-020 (резолв клиента)
- FR-030, 031 (транскрипция, аудит)
- FR-040, 041 (multi-tenancy)
- NFR-001 (latency p95 ≤1.5с), NFR-010, 011, NFR-020, 021, 022

Из native tools для MVP: `lookup_customer`, `check_calendar_slot`, `list_available_slots`, `book_appointment`, `transfer_to_human`, `end_call`.
Из MCP — google-calendar для внешних календарей мастеров.

Фаза 2: outbound calls, GraphRAG-память, мультиязычность.
Фаза 3: запись звонков, sentiment, расширенный admin UI.
