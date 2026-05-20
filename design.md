# Design: Voice Agent (Taskroom Reception)

## 1. Контекст и цели

Голосовой агент-ресепшионист для Taskroom. Принимает входящие звонки клиентов франчайзи (спа, салоны, школы), общается на нативном голосе через Gemini Live API, дёргает инструменты в реальном времени (календарь, CRM, БД клиентов), при необходимости сам инициирует исходящие звонки (владельцу локации, клиенту с напоминанием).

Главные принципы:
- Native real-time audio (без STT→LLM→TTS пайплайна) — Gemini Live принимает PCM и отдаёт PCM напрямую.
- Любой инструмент — это либо нативная функция Go, либо MCP-сервер. Добавление нового MCP — конфигурационная операция, не код.
- Один процесс (модульный монолит на Go) — переиспользуем стек Taskroom (gqlgen, Postgres+pgvector, multi-tenant).
- Все звонки и tool calls персистятся в Postgres, аудио — в S3-совместимое хранилище.

Не цели на этой итерации: видео-вход; WebRTC для браузеров (только телефония через Twilio); массовые обзвоны; multi-agent orchestration.

## 2. Архитектура верхнего уровня



┌─────────┐    PSTN     ┌──────────┐  WSS    ┌─────────────────────┐  WSS   ┌─────────────┐
│ Caller  │────────────▶│  Twilio  │────────▶│  Voice Agent (Go)   │───────▶│ Gemini Live │
└─────────┘             │  Voice   │         │                     │        │   API       │
                        └──────────┘         │  ┌───────────────┐  │        └─────────────┘
                             ▲               │  │ Session Mgr   │  │
                             │ TwiML         │  │ Audio Bridge  │  │
                             │ (outbound)    │  │ Tool Router   │  │
                             └───────────────│  │ MCP Hub       │  │
                                             │  └───────────────┘  │
                                             └──────┬──────────────┘
                                                    │
┌──────────────────────────┼──────────────────────────┐
▼                          ▼                          ▼
┌──────────────┐         ┌────────────────┐         ┌──────────────┐
│  Postgres    │         │  MCP Servers   │         │  Internal    │
│  + pgvector  │         │  (stdio/HTTP)  │         │  Tools (Go)  │
│  - calls     │         │  - Google Cal  │         │  - CRM       │
│  - sessions  │         │  - Gmail       │         │  - Customers │
│  - tool_logs │         │  - Weather     │         │  - Outbound  │
│  - memory    │         │  - Search      │         │              │
└──────────────┘         └────────────────┘         └──────────────┘


Поток входящего звонка:
1. Клиент звонит на номер локации Twilio.
2. Twilio дёргает `POST /webhooks/twilio/voice/incoming` — отвечаем TwiML с `<Connect><Stream>` на наш WSS endpoint.
3. Twilio открывает bidirectional WebSocket и шлёт μ-law 8kHz фреймы.
4. Voice Agent открывает второй WebSocket в Gemini Live API, передаёт system prompt + список инструментов (function declarations + MCP-инструменты).
5. Audio Bridge конвертирует μ-law 8kHz ↔ PCM 16-bit 16kHz (вход в Gemini) и PCM 24kHz → μ-law 8kHz (выход в Twilio).
6. Когда Gemini шлёт `toolCall`, Tool Router маршрутизирует вызов: либо нативный Go-хендлер, либо MCP-сервер. Результат через `send_tool_response`.
7. Транскрипты пишутся в `call_transcripts`. Tool calls — в `tool_invocations`.
8. На `stop` event от Twilio — закрываем сессию Gemini, сохраняем итоговый call record.

Поток исходящего звонка:
1. Триггер (напоминание, эскалация, ответ инструмента) → агент вызывает `outbound_call(to, reason, context)`.
2. Tool handler создаёт `outbound_call_request` в БД, идёт в Twilio REST API: `POST /Calls` с url нашего outbound webhook.
3. Twilio звонит абоненту. На pickup — тот же flow что для входящего, но system prompt инициализируется с контекстом ("ты звонишь Ивану, чтобы напомнить о записи на 15:00").
4. Если no-answer / busy / voicemail — Twilio шлёт `CallStatus` callback, агент логирует и опционально пробует позже.

## 3. Технологические решения и обоснования

### 3.1. Почему модульный монолит на Go
Стек уже выбран для Taskroom. Голосовой агент — один из бэкенд-модулей: `internal/voiceagent/`. Делит с остальной системой Postgres, GraphQL API (для UI наблюдения), Twilio credentials, multi-tenant resolver.

### 3.2. Почему Gemini Live напрямую
Pipecat/LiveKit добавляют слой между Twilio и Gemini → +50–150мс latency и лишняя зависимость. Native audio Gemini уже умеет barge-in, аффективный диалог, шумодав. Прямая интеграция = полный контроль над session state, tool routing, мультитенантностью.

### 3.3. Почему Twilio Media Streams, а не ConversationRelay
ConversationRelay делает свой STT/TTS и отдаёт текст, мы теряем native audio Gemini (тоны, паузы, эмоции). Media Streams отдаёт сырой μ-law — то, что нам нужно.

### 3.4. Audio resampling: μ-law 8kHz ↔ PCM 16-bit
Twilio шлёт `audio/x-mulaw;rate=8000` base64-кодированными фреймами по ~20мс. Gemini ждёт `audio/pcm;rate=16000` little-endian на вход и отдаёт 24kHz на выход.

Pipeline:
- Inbound (Twilio → Gemini): base64 decode → μ-law decode (G.711 lookup table) → upsample 8k→16k (linear interpolation достаточно для речи) → PCM16 LE → отправка как `Blob(mime_type="audio/pcm;rate=16000")`.
- Outbound (Gemini → Twilio): Gemini шлёт PCM16 24kHz → downsample 24k→8k → μ-law encode → base64 → Twilio `media` event.

Используем Go-библиотеку `github.com/zaf/g711` для μ-law, ресемплинг — самописный без зависимостей.

### 3.5. Tool system: нативные + MCP
Два класса инструментов с единым интерфейсом для Gemini.

Native tools (Go): прямые вызовы внутренних модулей Taskroom — booking, customer lookup, outbound call. Низкая latency (~5–20мс), не покидают процесс.

MCP tools: внешние интеграции через Model Context Protocol. Транспорты:
- stdio process (локальные серверы: Google Calendar MCP, Gmail MCP) — spawn при старте процесса, переиспользуется между сессиями.
- HTTP/SSE (удалённые MCP) — connection pool.

MCP Hub при старте процесса:
1. Читает `config/mcp.yaml` — список серверов с type/command/url/auth.
2. Для каждого делает `initialize` → `tools/list` → получает schemas.
3. Регистрирует инструменты в общем `ToolRegistry`.
4. Преобразует MCP schemas в Gemini `FunctionDeclaration`.

При вызове из Gemini:
1. `ToolRouter.Invoke(name, args)` ищет в реестре, понимает native или MCP.
2. Для MCP: `tools/call` через соответствующий транспорт.
3. Результат → JSON-serialise → `FunctionResponse` в Gemini Live.

Добавление нового MCP = запись в `mcp.yaml` + рестарт. Без правок Go-кода.

### 3.6. Tool behavior: BLOCKING vs NON_BLOCKING
Gemini Live поддерживает `behavior: NON_BLOCKING` для function declarations.
- BLOCKING (по умолчанию): быстрые tools (<500мс) — `lookup_customer`, `check_calendar_slot`. Модель ждёт ответа.
- NON_BLOCKING: долгие tools (>1с) — `send_email`, `outbound_call`. Модель продолжает говорить с клиентом ("сейчас отправлю, секунду"), результат прилетает позже с `scheduling: INTERRUPT` или `WHEN_IDLE`.

### 3.7. Multi-tenancy
Каждый Twilio-номер привязан к `location_id`. При входящем:
1. Из webhook вынимаем `To`-номер → резолвим `location_id` → `tenant_id`.
2. Загружаем конфиг локации: язык, system prompt overrides, доступные инструменты, рабочие часы, fallback-маршрут.
3. Все tool calls идут с `tenant_id` в контексте — строгая изоляция данных.

### 3.8. Память между звонками (GraphRAG)
Переиспользуем GraphRAG-движок Taskroom. При старте сессии:
1. Резолвим клиента по `From` (caller_id) → если есть `customer_id`, подгружаем последние N взаимодействий.
2. Embedding query: "Краткая сводка известного о клиенте" → 3–5 топ-релевантных фрагментов.
3. Инжектим в system prompt.
4. После звонка фоновый джоб суммаризует транскрипт, обновляет память.

### 3.9. Прерывания и barge-in
Gemini Live нативно поддерживает barge-in через `interrupted` флаг в `server_content`. При его получении:
1. Дропаем все буферизованные исходящие PCM-фреймы в Twilio.
2. Шлём Twilio `clear` event (очищает его playout buffer).
3. Продолжаем стримить новое аудио от Gemini.

### 3.10. Безопасность
- Twilio signature verification: все webhook-эндпоинты валидируют `X-Twilio-Signature` HMAC. Без подписи — 403.
- WebSocket auth от Twilio: в TwiML `<Stream>` передаём `<Parameter name="auth_token">` с одноразовым токеном (TTL 60с), валидируем в WS handshake.
- Gemini API key хранится в HashiCorp Vault.
- PII в транскриптах: опциональная regex-маскировка номеров карт и SSN.
- Запись звонков только при явном согласии в начале + согласие клиента (либо если законодательство позволяет).

### 3.11. Наблюдаемость
- Структурированные логи (zerolog): `call_id`, `session_id`, `tenant_id` в каждой записи.
- Prometheus метрики: `voice_session_duration_seconds`, `tool_invocation_duration_seconds{name}`, `gemini_ws_reconnects_total`, `audio_buffer_underrun_total`, `mcp_call_errors_total{server}`.
- OpenTelemetry tracing: спан на сессию, дочерние спаны на каждый tool call.
- Live-дашборд активных звонков в админке Taskroom через GraphQL subscription.

## 4. Структура пакетов Go



internal/voiceagent/
├── server/
│   ├── twilio_webhook.go       # HTTP handlers: /webhooks/twilio/voice/{incoming,outbound,status}
│   ├── twilio_ws.go            # WS handler для Media Streams
│   └── twiml.go                # TwiML builder
├── session/
│   ├── manager.go              # Реестр активных сессий, lifecycle
│   ├── session.go              # State machine одной сессии
│   └── context.go              # Per-session ctx с tenant_id, call_id, customer_id
├── gemini/
│   ├── client.go               # WS-клиент к Gemini Live
│   ├── config.go               # Сборка LiveConnectConfig (system prompt, tools, voice)
│   ├── codec.go                # Парсинг serverContent, toolCall, transcription
│   └── reconnect.go            # Backoff и session resumption
├── audio/
│   ├── mulaw.go                # μ-law encode/decode
│   ├── resample.go             # 8k↔16k, 24k→8k linear interpolation
│   └── jitter_buffer.go        # Output jitter buffer для Twilio
├── tools/
│   ├── registry.go             # ToolRegistry: native + MCP unified
│   ├── router.go               # Маршрутизация вызовов
│   ├── declarations.go         # MCP schema → Gemini FunctionDeclaration
│   └── native/
│       ├── calendar.go         # book_appointment, check_slot, list_slots
│       ├── customer.go         # lookup_customer, create_customer, update_customer
│       ├── crm.go              # log_interaction, get_history
│       ├── outbound.go         # outbound_call (агент звонит сам)
│       ├── handoff.go          # transfer_to_human
│       └── memory.go           # query_memory, save_note
├── mcp/
│   ├── hub.go                  # Менеджер MCP-серверов
│   ├── stdio_transport.go      # stdio MCP servers
│   ├── http_transport.go       # HTTP/SSE MCP servers
│   ├── config.go               # Парсинг mcp.yaml
│   └── proxy.go                # MCP server как tool в Gemini
├── persistence/
│   ├── calls.go                # CRUD таблицы calls
│   ├── transcripts.go          # call_transcripts
│   ├── tool_logs.go            # tool_invocations
│   └── recordings.go           # S3 upload, метаданные
├── outbound/
│   ├── initiator.go            # Twilio REST API: исходящий звонок
│   └── queue.go                # Очередь outbound (rate limit, retry)
└── config/
├── tenant.go               # Tenant-specific конфиги
└── prompts.go              # System prompt templates per tenant/agent type


## 5. Схема БД (дополнения к Taskroom)

```sql
CREATE TABLE calls (
  id                  UUID PRIMARY KEY,
  tenant_id           UUID NOT NULL REFERENCES tenants(id),
  location_id         UUID NOT NULL REFERENCES locations(id),
  twilio_call_sid     TEXT UNIQUE NOT NULL,
  direction           TEXT NOT NULL CHECK (direction IN ('inbound','outbound')),
  from_number         TEXT NOT NULL,
  to_number           TEXT NOT NULL,
  customer_id         UUID REFERENCES customers(id),
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
CREATE INDEX idx_calls_tenant_started ON calls(tenant_id, started_at DESC);
CREATE INDEX idx_calls_customer ON calls(customer_id) WHERE customer_id IS NOT NULL;

CREATE TABLE call_transcripts (
  id              UUID PRIMARY KEY,
  call_id         UUID NOT NULL REFERENCES calls(id) ON DELETE CASCADE,
  speaker         TEXT NOT NULL CHECK (speaker IN ('user','agent','system')),
  content         TEXT NOT NULL,
  ts_offset_ms    INT NOT NULL,
  is_final        BOOL NOT NULL DEFAULT true,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_transcripts_call ON call_transcripts(call_id, ts_offset_ms);

CREATE TABLE tool_invocations (
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
CREATE INDEX idx_tools_call ON tool_invocations(call_id, ts_offset_ms);

CREATE TABLE outbound_call_requests (
  id              UUID PRIMARY KEY,
  tenant_id       UUID NOT NULL REFERENCES tenants(id),
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

CREATE TABLE mcp_servers (
  id              UUID PRIMARY KEY,
  tenant_id       UUID REFERENCES tenants(id),  -- NULL = глобальный
  name            TEXT NOT NULL,
  transport       TEXT NOT NULL CHECK (transport IN ('stdio','http','sse')),
  config          JSONB NOT NULL,
  enabled         BOOL NOT NULL DEFAULT true,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(tenant_id, name)
);
```

## 6. Конфигурация MCP

config/mcp.yaml:

```yaml
servers:
  - name: google-calendar
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-google-calendar"]
    env:
      GOOGLE_CREDENTIALS_PATH: /etc/taskroom/google-creds.json
    enabled: true

  - name: gmail
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-gmail"]
    enabled: true

  - name: weather
    transport: http
    url: https://weather-mcp.example.com/sse
    auth:
      type: bearer
      token_env: WEATHER_MCP_TOKEN
    enabled: true

  - name: web-search
    transport: stdio
    command: npx
    args: ["-y", "@modelcontextprotocol/server-brave-search"]
    env:
      BRAVE_API_KEY_ENV: BRAVE_API_KEY
    enabled: true
```

## 7. Системный промпт (скелет для ресепшна)

Ты — голосовой ресепшионист {{ .Location.Name }}, {{ .Location.BusinessType }}.

РОЛЬ:
- Принимаешь звонки клиентов.
- Записываешь на услуги (check_calendar_slot и book_appointment).
- Отвечаешь на вопросы о часах работы, услугах, ценах.
- Если клиент известен — обращайся по имени.
- Если вне твоей компетенции — transfer_to_human.
- Если нужно подтверждение от владельца — outbound_call.

СТИЛЬ:
- Говори на {{ .Location.Language }}.
- Кратко, дружелюбно, профессионально. Не больше 2-3 предложений за реплику.
- Если перебивают — остановись, слушай.
- При записи проговаривай вслух: услугу, дату, время, имя — для подтверждения.

КОНТЕКСТ ЛОКАЦИИ:
{{ .Location.Hours }}
Услуги: {{ .Location.Services }}

{{ if .Customer }}
КЛИЕНТ:
Имя: {{ .Customer.Name }}
Последние взаимодействия: {{ .Customer.RecentSummary }}
{{ end }}

ПРАВИЛА:
- Не выдумывай услуги/цены/слоты. Сначала вызови инструмент.
- При ошибке инструмента — извинись и предложи альтернативу.
- В конце разговора подтверди: "Записал тебя на… Жду в…"


## 8. Каталог инструментов (native, первая итерация)



|Инструмент            |Behavior    |Назначение                                          |
|----------------------|------------|----------------------------------------------------|
|lookup_customer       |BLOCKING    |Поиск клиента по телефону / имени.                  |
|create_customer       |BLOCKING    |Создать клиента, если новый.                        |
|check_calendar_slot   |BLOCKING    |Свободен ли слот у мастера/услуги в дату/время.     |
|list_available_slots  |BLOCKING    |Список свободных слотов на день/неделю.             |
|book_appointment      |BLOCKING    |Создать запись.                                     |
|cancel_appointment    |BLOCKING    |Отменить запись.                                    |
|reschedule_appointment|BLOCKING    |Перенести запись.                                   |
|get_customer_history  |BLOCKING    |История посещений клиента.                          |
|log_interaction       |NON_BLOCKING|Записать заметку о звонке (автоматически в конце).  |
|query_memory          |BLOCKING    |GraphRAG-запрос по памяти локации/клиента.          |
|save_note             |NON_BLOCKING|Сохранить факт в долгосрочную память.               |
|outbound_call         |NON_BLOCKING|Инициировать исходящий звонок (владельцу / клиенту).|
|transfer_to_human     |BLOCKING    |Перевести на оператора (Twilio `<Dial>`).           |
|send_sms              |NON_BLOCKING|SMS (подтверждение, ссылка).                        |
|end_call              |BLOCKING    |Корректно завершить звонок.                         |

## 9. Управление состоянием сессии

```go
type SessionState struct {
    CallID        uuid.UUID
    TwilioCallSID string
    TenantID      uuid.UUID
    LocationID    uuid.UUID
    CustomerID    *uuid.UUID

    StreamSID     string
    GeminiConn    *gemini.Conn
    TwilioConn    *websocket.Conn

    StartedAt     time.Time
    AnsweredAt    *time.Time
    State         SessionPhase // ringing → answered → talking → tool_call → ending → ended

    TranscriptBuf *transcriptBuffer
    AudioOutBuf   *jitter.Buffer
    ToolsInFlight map[string]*ToolCall

    mu sync.Mutex
}
```

## 10. Failure modes и обработка



|Сценарий                        |Поведение                                                                                 |
|--------------------------------|------------------------------------------------------------------------------------------|
|Gemini WS drop в середине звонка|Reconnect с `session_resumption_handle`, иначе — transfer_to_human или TwiML с сообщением.|
|Twilio WS drop                  |Считаем звонок завершённым, помечаем failed, закрываем Gemini.                            |
|MCP не отвечает на tool call    |Timeout 8с → возвращаем модели error → она извиняется и пробует обходное решение.         |
|Native tool panic               |Recover middleware → error в Gemini → лог + alert.                                        |
|Audio buffer underrun           |Шлём в Twilio comfort noise → метрика +1.                                                 |
|Пользователь молчит >15с        |Агент сам спрашивает “Вы здесь?” → ещё 15с тишины → end_call.                             |
|Превышен лимит длины (15 мин)   |Агент проактивно завершает / переводит на оператора.                                      |

## 11. Открытые вопросы для партнёра

	•	Voice ID Gemini per локация/язык (Aoede, Charon, Kore, Fenrir, Puck, Zephyr — зависит от модели на момент имплементации).
	•	Согласие на запись — юридический вопрос по юрисдикции.
	•	Warm vs cold transfer на оператора.
	•	Биллинг: учёт минут Twilio и токенов Gemini per-tenant.
