import os
import yaml
import pyaudio

# File paths
SETTINGS_DIR = "settings"
USER_STORE_DIR = "user_store"

MEMORY_FILE = os.path.join(USER_STORE_DIR, "memory.md")
TRANSCRIPT_FILE = os.path.join(USER_STORE_DIR, "transcript.md")
CONFIG_FILE = os.path.join(SETTINGS_DIR, "config.yaml")
DEBUG_LOG_FILE = os.path.join(USER_STORE_DIR, "debug_log.txt")

os.makedirs(USER_STORE_DIR, exist_ok=True)

def load_config():
    """Loads application configuration from YAML file."""
    if os.path.exists(CONFIG_FILE):
        with open(CONFIG_FILE, "r", encoding="utf-8") as f:
            try:
                return yaml.safe_load(f)
            except yaml.YAMLError as exc:
                print(f"Error parsing YAML: {exc}")
    return {}

def save_to_env(key, value):
    """Saves a setting to the .env file."""
    env_file = ".env"
    lines = []
    if os.path.exists(env_file):
        with open(env_file, "r", encoding="utf-8") as f:
            lines = f.readlines()

    found = False
    for i, line in enumerate(lines):
        if line.strip().startswith(f"{key}="):
            lines[i] = f"{key}={value}\n"
            found = True
            break

    if not found:
        if lines and not lines[-1].endswith("\n"):
            lines.append("\n")
        lines.append(f"{key}={value}\n")

    with open(env_file, "w", encoding="utf-8") as f:
        f.writelines(lines)
    print(f"[Saved to .env: {key}={value}]")

def select_devices_and_prefs(pya_instance, current_config):
    """Handles device selection and preferences, saving to .env if needed."""

    model_settings = current_config.get('model', {})
    devices_config = current_config.get('devices', {})

    # Voice selection — Priority: YAML > Env
    voices = ["Zephyr", "Aoede", "Charon", "Fenrir", "Kore", "Puck"]
    voice_name = model_settings.get("voice") or os.environ.get("VOICE_NAME")

    if voice_name not in voices:
        print("\n--- Select Voice ---")
        for i, v in enumerate(voices):
            print(f"[{i}] {v}")
        v_idx = int(input("Enter voice number (default 0): ") or 0)
        voice_name = voices[v_idx] if v_idx < len(voices) else "Zephyr"
        save_to_env("VOICE_NAME", voice_name)
    print(f"[Voice: {voice_name}]")

    # Speed selection — Priority: YAML > Env
    speeds = ["normal", "fast", "slow"]
    speed = model_settings.get("speed") or os.environ.get("SPEED")

    if speed not in speeds:
        print("\n--- Select Speed ---")
        for i, s in enumerate(speeds):
            print(f"[{i}] {s}")
        s_idx = int(input("Enter speed number (default 0): ") or 0)
        speed = speeds[s_idx] if s_idx < len(speeds) else "normal"
        save_to_env("SPEED", speed)
    print(f"[Speed: {speed}]")

    # Audio Devices
    info = pya_instance.get_host_api_info_by_index(0)
    num_devices = info.get('deviceCount')
    inputs = []
    outputs = []
    for i in range(num_devices):
        device_info = pya_instance.get_device_info_by_host_api_device_index(0, i)
        if device_info.get('maxInputChannels') > 0:
            inputs.append({"id": i, "name": device_info.get('name')})
        if device_info.get('maxOutputChannels') > 0:
            outputs.append({"id": i, "name": device_info.get('name')})

    # Input Device — Priority: YAML > Env
    env_input = devices_config.get("input") or os.environ.get("INPUT_DEVICE_NAME")
    found_input = next((d for d in inputs if d["name"] == env_input), None)
    if found_input:
        input_device_index = found_input["id"]
        print(f"[Microphone: {env_input}]")
    else:
        print("\n--- Available Input Devices (Microphones) ---")
        for i, d in enumerate(inputs):
            print(f"[{i}] {d['name']}")
        idx = int(input("\nSelect microphone number (default 0): ") or 0)
        selected = inputs[idx] if idx < len(inputs) else inputs[0]
        input_device_index = selected["id"]
        save_to_env("INPUT_DEVICE_NAME", selected["name"])

    # Output Device — Priority: YAML > Env
    env_output = devices_config.get("output") or os.environ.get("OUTPUT_DEVICE_NAME")
    found_output = next((d for d in outputs if d["name"] == env_output), None)
    if found_output:
        output_device_index = found_output["id"]
        print(f"[Speaker: {env_output}]")
    else:
        print("\n--- Available Output Devices (Speakers) ---")
        for i, d in enumerate(outputs):
            print(f"[{i}] {d['name']}")
        idx = int(input("\nSelect speaker number (default 0): ") or 0)
        selected = outputs[idx] if idx < len(outputs) else outputs[0]
        output_device_index = selected["id"]
        save_to_env("OUTPUT_DEVICE_NAME", selected["name"])

    return {
        "voice_name": voice_name,
        "speed": speed,
        "input_device_index": input_device_index,
        "output_device_index": output_device_index
    }
