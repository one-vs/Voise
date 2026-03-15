# Professional Configuration Guide

This project uses a unified YAML configuration file located at `settings/config.yaml`. This approach allows for structured, easily parsable settings that separate model parameters from AI persona and system rules.

## File Structure: `settings/config.yaml`

The configuration is divided into three main sections:

### 1. `model`

Controls everything related to the Gemini model and the audio/video stream.

- **`name`**: The model identifier (e.g., `models/gemini-2.5-flash-native-audio-preview-12-2025`).
- **`voice`**: Prebuilt voice name (`Zephyr`, `Aoede`, etc.).
- **`speed`**: Default speech rate (`normal`, `fast`, `slow`).
- **`audio_pipeline`**:
  - `silence_duration_ms`: Wait time before closing user turn.
  - `start_of_speech_sensitivity`: Trigger sensitivity for user voice.
  - `end_of_speech_sensitivity`: Sensitivity for user stopping.
- **`media`**: Resolution settings.
- **`context`**: Token thresholds for sliding window compression.

### 2. `instructions`

Contains the actual prompts used by the AI.

- **`personality`**: Who the AI is and how it should speak (identity).
- **`system_rules`**: Technical rules (tool usage, safety, "thought blocks" prevention). Includes the `{{SPEED_INSTRUCTION}}` placeholder, which is automatically filled by the app based on the `speed` setting.

### 3. `devices`

Hardware preferences.

- **`input`**: Exact name of the microphone device.
- **`output`**: Exact name of the speaker device.

---

## Why YAML?

Using YAML is the industry standard for configuration because:

1. **Structure**: Allows nested objects instead of flat lists.
2. **Readability**: Supports multi-line strings (using the `|` symbol) which is perfect for long AI prompts.
3. **Data Types**: Correctly handles booleans, integers, and strings without manual conversion in code.
