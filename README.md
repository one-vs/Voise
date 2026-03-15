# Voise — Real-time AI Voice Assistant powered by Gemini Live API

An interactive AI assistant with real-time voice and text communication, persistent memory, Google Search, and support for camera/screen streaming.

## Features

- **Real-time Voice** — zero-latency conversation via Gemini Live API
- **Google Search** — AI can search the web during conversation
- **Persistent Memory** — AI remembers facts about you between sessions
- **Modular Tools** — add new AI capabilities by dropping a file into `tools/`
- **Single Config** — model, voice, speed, and AI persona in one YAML file
- **Docker Support** — run in an isolated container

## Project Structure

```
voise/
├── ai_studio_code.py       # Main application entrypoint
├── config_utils.py         # Config loading, device selection
├── settings/
│   ├── config.yaml         # Main config (model, instructions, devices)
│   └── parameters_guide.md # Parameter reference
├── tools/                  # Auto-loaded AI tools
│   ├── save_user_memory.py
│   ├── read_user_memory.py
│   └── update_user_memory.py
├── user_store/             # Runtime data (gitignored)
│   ├── memory.md           # Persistent AI memory
│   ├── transcript.md       # Session transcripts
│   └── debug_log.txt       # Debug output (--debug mode)
├── Dockerfile
└── docker-compose.yaml
```

## Quick Start

### Prerequisites

- Python 3.12+
- A Gemini API key — get one free at [aistudio.google.com](https://aistudio.google.com/apikey)

---

### Windows

#### Option A — with `uv` (recommended)

1. Install [uv](https://docs.astral.sh/uv/getting-started/installation/):
   ```powershell
   powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"
   ```

2. Clone the repo and enter the folder:
   ```powershell
   git clone <repo-url>
   cd voise
   ```

3. Create `.env` from the template and add your key:
   ```powershell
   copy .env.example .env
   notepad .env
   ```
   Set `GEMINI_API_KEY=your_key_here`

4. Install dependencies and run:
   ```powershell
   uv sync
   uv run python ai_studio_code.py
   ```

#### Option B — plain Python

1. Install [Python 3.12+](https://www.python.org/downloads/) (check "Add to PATH")

2. Install system dependency for PyAudio:
   ```powershell
   pip install pipwin
   pipwin install pyaudio
   ```

3. Install the rest:
   ```powershell
   pip install google-genai opencv-python pillow mss pyyaml python-dotenv
   ```

4. Copy `.env.example` to `.env` and set your API key, then run:
   ```powershell
   python ai_studio_code.py
   ```

---

### macOS

#### Option A — with `uv` (recommended)

1. Install [uv](https://docs.astral.sh/uv/getting-started/installation/):
   ```bash
   curl -LsSf https://astral.sh/uv/install.sh | sh
   ```

2. Clone the repo and enter the folder:
   ```bash
   git clone <repo-url>
   cd voise
   ```

3. Create `.env` from the template and add your key:
   ```bash
   cp .env.example .env
   open -e .env   # or: nano .env
   ```
   Set `GEMINI_API_KEY=your_key_here`

4. Install PortAudio (required by PyAudio):
   ```bash
   brew install portaudio
   ```

5. Install dependencies and run:
   ```bash
   uv sync
   uv run python ai_studio_code.py
   ```

#### Option B — plain Python

1. Install [Python 3.12+](https://www.python.org/downloads/) or via Homebrew:
   ```bash
   brew install python@3.12
   ```

2. Install PortAudio and dependencies:
   ```bash
   brew install portaudio
   pip install google-genai opencv-python pyaudio pillow mss pyyaml python-dotenv
   ```

3. Copy `.env.example` to `.env` and set your API key, then run:
   ```bash
   python ai_studio_code.py
   ```

---

## CLI Options

```bash
# Default — voice only
uv run python ai_studio_code.py

# Stream from camera
uv run python ai_studio_code.py --mode camera

# Stream from screen
uv run python ai_studio_code.py --mode screen

# Enable debug logging to user_store/debug_log.txt
uv run python ai_studio_code.py --debug
```

On first run, the app will ask you to select your microphone and speaker. The choices are saved to `.env` automatically.

---

## Configuration

All settings live in **`settings/config.yaml`**. Key options:

| Key | Description |
|---|---|
| `model.voice` | AI voice: `Zephyr`, `Puck`, `Aoede`, `Charon`, `Fenrir`, `Kore` |
| `model.speed` | Speech rate: `normal`, `fast`, `slow` |
| `instructions.personality` | Who the AI is and how it speaks |
| `instructions.greeting` | First message sent to the AI on startup |
| `devices.input` | Microphone name (auto-detected on first run) |
| `devices.output` | Speaker name (auto-detected on first run) |

See `settings/parameters_guide.md` for the full reference.

---

## Adding Custom Tools

Drop a new file into `tools/` — it will be auto-loaded on next start.

Each tool file must export:
- A function named after the file (e.g. `my_tool.py` → `def my_tool(...)`)
- A `declaration` dict with the JSON schema for the Gemini API

**Example** — `tools/get_weather.py`:
```python
def get_weather(city: str):
    # your implementation
    return {"temperature": "22°C", "condition": "Sunny"}

declaration = {
    "name": "get_weather",
    "description": "Returns current weather for a given city.",
    "parameters": {
        "type": "OBJECT",
        "properties": {
            "city": {"type": "STRING", "description": "City name"}
        },
        "required": ["city"]
    }
}
```

---

## Docker

```bash
# Build
docker build -t voise-app .

# Run with Docker Compose
docker-compose up
```

> **Note:** Audio and video passthrough in Docker requires additional setup (PulseAudio or device permissions on Linux). Native run is recommended for development.
