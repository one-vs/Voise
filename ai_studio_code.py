import os
import asyncio
import base64
import datetime
import io
import json
import traceback
from dotenv import load_dotenv
import config_utils
import tools

load_dotenv()

import cv2
import pyaudio
import PIL.Image

import argparse

from google import genai
from google.genai import types

FORMAT = pyaudio.paInt16
CHANNELS = 1
SEND_SAMPLE_RATE = 16000
RECEIVE_SAMPLE_RATE = 24000
CHUNK_SIZE = 1024

# Default configuration
DEFAULT_MODEL = "models/gemini-2.5-flash-native-audio-preview-12-2025"
DEFAULT_MODE = "none"

client = genai.Client(
    http_options={"api_version": "v1beta"},
    api_key=os.environ.get("GEMINI_API_KEY"),
)


def log_transcript(speaker, text):
    """Writes dialog to the transcript file."""
    try:
        now = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        with open(config_utils.TRANSCRIPT_FILE, "a", encoding="utf-8") as f:
            f.write(f"[{now}] {speaker}: {text}\n")
            f.flush()
    except Exception as e:
        print(f"[Transcript write error]: {e}")

def clean_for_log(obj):
    """Recursively replaces bytes with descriptions for JSON logging."""
    if isinstance(obj, dict):
        return {k: clean_for_log(v) for k, v in obj.items()}
    elif isinstance(obj, list):
        return [clean_for_log(item) for item in obj]
    elif isinstance(obj, bytes):
        return f"<BYTES: {len(obj)} bytes>"
    return obj


def get_config(voice_name="Zephyr", speed="normal", app_config=None):
    if app_config is None:
        app_config = {}

    model_config = app_config.get("model", {})
    instruction_config = app_config.get("instructions", {})

    # Combine personality and system rules
    personality = instruction_config.get("personality", "").strip()
    system_rules = instruction_config.get("system_rules", "").strip()
    
    system_instruction = f"{personality}\n\n{system_rules}"
    
    # Handle speed instruction replacement
    speed_phrases = instruction_config.get("speed_phrases", {})
    speed_text = speed_phrases.get(speed, "")
    
    if not speed_text:
        # Fallback if phrases missing in YAML
        if speed == "fast":
            speed_text = "Please speak at an accelerated pace."
        elif speed == "slow":
            speed_text = "Please speak slowly and steadily."
    
    if "{{SPEED_INSTRUCTION}}" in system_instruction:
        system_instruction = system_instruction.replace("{{SPEED_INSTRUCTION}}", speed_text)
    else:
        system_instruction += f"\n{speed_text}"

    audio_pipeline = model_config.get("audio_pipeline", {})
    media_config = model_config.get("media", {})
    context_config = model_config.get("context", {})

    return types.LiveConnectConfig(
        response_modalities=["AUDIO"],
        system_instruction=types.Content(parts=[types.Part(text=system_instruction)]),
        media_resolution=media_config.get("resolution", "MEDIA_RESOLUTION_MEDIUM"),
        input_audio_transcription=types.AudioTranscriptionConfig(),
        output_audio_transcription=types.AudioTranscriptionConfig(),
        tools=[
            types.Tool(google_search=types.GoogleSearch()),
            types.Tool(function_declarations=tools.TOOL_DECLARATIONS),
        ],
        speech_config=types.SpeechConfig(
            voice_config=types.VoiceConfig(
                prebuilt_voice_config=types.PrebuiltVoiceConfig(voice_name=voice_name)
            )
        ),
        realtime_input_config=types.RealtimeInputConfig(
            automatic_activity_detection=types.AutomaticActivityDetection(
                silence_duration_ms=int(audio_pipeline.get("silence_duration_ms", 1000)),
                start_of_speech_sensitivity=audio_pipeline.get("start_of_speech_sensitivity", 'START_SENSITIVITY_HIGH'),
                end_of_speech_sensitivity=audio_pipeline.get("end_of_speech_sensitivity", 'END_SENSITIVITY_LOW'),
            )
        ),
        context_window_compression=types.ContextWindowCompressionConfig(
            trigger_tokens=int(context_config.get("trigger_tokens", 104857)),
            sliding_window=types.SlidingWindow(target_tokens=int(context_config.get("target_tokens", 52428))),
        ),
    )

pya = pyaudio.PyAudio()


class AudioLoop:
    def __init__(self, video_mode=DEFAULT_MODE, debug=False):
        self.video_mode = video_mode
        self.debug = debug

        self.audio_in_queue = None
        self.out_queue = None

        self.session = None

        self.audio_stream = None
        self.input_device_index = None
        self.output_device_index = None
        self.voice_name = "Zephyr"
        self.speed = "normal"
        self.memory = ""
        self.app_config = {}
        self.model_name = DEFAULT_MODEL
        self.current_speaker = None
        self.current_text = ""

    def _flush_transcript(self):
        if self.current_speaker and self.current_text.strip():
            log_transcript(self.current_speaker, self.current_text.strip())
        self.current_text = ""
        self.current_speaker = None

    def _load_files(self):
        # Memory
        if os.path.exists(config_utils.MEMORY_FILE):
            with open(config_utils.MEMORY_FILE, "r", encoding="utf-8") as f:
                self.memory = f.read().strip()
                print("[User memory loaded]")

        # Application config (YAML)
        self.app_config = config_utils.load_config()
        if self.app_config:
            print("[Project configuration loaded (YAML)]")
            model_settings = self.app_config.get("model", {})
            self.model_name = model_settings.get("name", DEFAULT_MODEL)
            self.voice_name = model_settings.get("voice", "Zephyr")
            self.speed = model_settings.get("speed", "normal")

    def _select_devices_and_prefs(self):
        self._load_files()
        
        prefs = config_utils.select_devices_and_prefs(pya, self.app_config)
        
        self.voice_name = prefs["voice_name"]
        self.speed = prefs["speed"]
        self.input_device_index = prefs["input_device_index"]
        self.output_device_index = prefs["output_device_index"]

    async def send_text(self):
        while True:
            text = await asyncio.to_thread(
                input,
                "message > ",
            )
            if self.session is not None:
                log_transcript("You (Text)", text)
                await self.session.send_client_content(
                    turns=types.Content(parts=[types.Part(text=text or "")]),
                    turn_complete=True,
                )

    def _get_frame(self, cap):
        ret, frame = cap.read()
        # Check if the frame was read successfully
        if not ret:
            return None
        # Fix: Convert BGR to RGB color space
        # OpenCV captures in BGR but PIL expects RGB format
        # This prevents the blue tint in the video feed
        frame_rgb = cv2.cvtColor(frame, cv2.COLOR_BGR2RGB)
        img = PIL.Image.fromarray(frame_rgb)  # Now using RGB frame
        img.thumbnail((1024, 1024))

        image_io = io.BytesIO()
        img.save(image_io, format="jpeg")
        image_io.seek(0)

        mime_type = "image/jpeg"
        image_bytes = image_io.read()
        return {"mime_type": mime_type, "data": base64.b64encode(image_bytes).decode()}

    async def get_frames(self):
        # This takes about a second, and will block the whole program
        # causing the audio pipeline to overflow if you don't to_thread it.
        cap = await asyncio.to_thread(
            cv2.VideoCapture, 0
        )  # 0 represents the default camera

        while True:
            frame = await asyncio.to_thread(self._get_frame, cap)
            if frame is None:
                break

            await asyncio.sleep(1.0)

            if self.out_queue is not None:
                await self.out_queue.put(frame)

        # Release the VideoCapture object
        cap.release()

    def _get_screen(self):
        try:
            import mss  # pytype: disable=import-error # pylint: disable=g-import-not-at-top
        except ImportError as e:
            raise ImportError("Please install mss package using 'pip install mss'") from e
        sct = mss.mss()
        monitor = sct.monitors[0]

        i = sct.grab(monitor)

        mime_type = "image/jpeg"
        image_bytes = mss.tools.to_png(i.rgb, i.size)
        img = PIL.Image.open(io.BytesIO(image_bytes))

        image_io = io.BytesIO()
        img.save(image_io, format="jpeg")
        image_io.seek(0)

        image_bytes = image_io.read()
        return {"mime_type": mime_type, "data": base64.b64encode(image_bytes).decode()}

    async def get_screen(self):

        while True:
            frame = await asyncio.to_thread(self._get_screen)
            if frame is None:
                break

            await asyncio.sleep(1.0)

            if self.out_queue is not None:
                await self.out_queue.put(frame)

    async def send_realtime(self):
        while True:
            if self.out_queue is not None:
                msg = await self.out_queue.get()
                if self.session is not None:
                    if msg["mime_type"].startswith("audio/"):
                        await self.session.send_realtime_input(audio=msg)
                    else:
                        await self.session.send_realtime_input(media=msg)

    async def listen_audio(self):
        if self.input_device_index is None:
            mic_info = pya.get_default_input_device_info()
            self.input_device_index = mic_info["index"]

        self.audio_stream = await asyncio.to_thread(
            pya.open,
            format=FORMAT,
            channels=CHANNELS,
            rate=SEND_SAMPLE_RATE,
            input=True,
            input_device_index=self.input_device_index,
            frames_per_buffer=CHUNK_SIZE,
        )
        if __debug__:
            kwargs = {"exception_on_overflow": False}
        else:
            kwargs = {}
        while True:
            data = await asyncio.to_thread(self.audio_stream.read, CHUNK_SIZE, **kwargs)
            if self.out_queue is not None:
                await self.out_queue.put({"data": data, "mime_type": "audio/pcm"})

    async def receive_audio(self):
        """Background task: reads responses from the websocket, plays audio, and handles tool calls."""
        while True:
            if self.session is None:
                await asyncio.sleep(0)
                continue
            async for response in self.session.receive():
                if self.debug:
                    try:
                        resp_dict = response.model_dump()
                        cleaned_dict = clean_for_log(resp_dict)

                        with open(config_utils.DEBUG_LOG_FILE, "a", encoding="utf-8") as f:
                            now = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S.%f")
                            f.write(f"--- {now} ---\n")
                            f.write(json.dumps(cleaned_dict, ensure_ascii=False, indent=2) + "\n")
                            f.flush()
                    except Exception as e:
                        with open(config_utils.DEBUG_LOG_FILE, "a", encoding="utf-8") as f:
                            traceback.print_exc(file=f)
                            f.write(f"Error logging response: {e}\n")
                            f.flush()

                # Check for server content (transcriptions)
                if sc := response.server_content:
                    # We ignore sc.model_turn.parts to avoid English "thoughts" and other noise.
                    # We rely on the official transcription fields for clean spoken text.

                    if sc.input_transcription:
                        if self.current_speaker != "You":
                            self._flush_transcript()
                            self.current_speaker = "You"
                        # API sends chunks that may be parts of words — spaces are already in the data.
                        self.current_text += (sc.input_transcription.text or "")

                    if sc.output_transcription:
                        if self.current_speaker != "AI":
                            self._flush_transcript()
                            self.current_speaker = "AI"
                        self.current_text += (sc.output_transcription.text or "")

                    if sc.interrupted:
                        self._flush_transcript()
                        log_transcript("System", "Interruption")

                    if sc.turn_complete:
                        self._flush_transcript()

                if data := response.data:
                    self.audio_in_queue.put_nowait(data)
                    continue

                if tool_call := response.tool_call:
                    f_responses = []
                    for call in tool_call.function_calls:
                        name = call.name
                        args = call.args
                        try:
                            if name in tools.TOOL_MAP:
                                res = tools.TOOL_MAP[name](**args)
                            else:
                                res = {"status": "error", "message": f"Unknown function: {name}"}
                        except Exception as e:
                            print(f"Error executing tool {name}: {e}")
                            res = {"status": "error", "message": str(e)}

                        f_responses.append(
                            types.FunctionResponse(
                                name=name,
                                id=call.id,
                                response=res,
                            )
                        )

                    if f_responses:
                        await self.session.send_tool_response(function_responses=f_responses)

            # If the model is interrupted it sends a turn_complete.
            # Drain the audio queue so playback stops immediately.
            while not self.audio_in_queue.empty():
                self.audio_in_queue.get_nowait()

    async def play_audio(self):
        stream = await asyncio.to_thread(
            pya.open,
            format=FORMAT,
            channels=CHANNELS,
            rate=RECEIVE_SAMPLE_RATE,
            output=True,
            output_device_index=self.output_device_index,
        )
        while True:
            if self.audio_in_queue is not None:
                bytestream = await self.audio_in_queue.get()
                await asyncio.to_thread(stream.write, bytestream)

    async def run(self):
        try:
            self._select_devices_and_prefs()
            config = get_config(self.voice_name, self.speed, self.app_config)
            async with (
                client.aio.live.connect(model=self.model_name, config=config) as session,
                asyncio.TaskGroup() as tg,
            ):
                self.session = session

                self.audio_in_queue = asyncio.Queue()
                self.out_queue = asyncio.Queue(maxsize=5)

                # Greeting on start
                greeting = self.app_config.get("instructions", {}).get("greeting", "Hello!")
                await self.session.send_client_content(
                    turns=types.Content(parts=[types.Part(text=greeting)]),
                    turn_complete=True,
                )

                send_text_task = tg.create_task(self.send_text())
                tg.create_task(self.send_realtime())
                tg.create_task(self.listen_audio())
                if self.video_mode == "camera":
                    tg.create_task(self.get_frames())
                elif self.video_mode == "screen":
                    tg.create_task(self.get_screen())

                tg.create_task(self.receive_audio())
                tg.create_task(self.play_audio())

                await send_text_task
                raise asyncio.CancelledError("User requested exit")

        except asyncio.CancelledError:
            pass
        except ExceptionGroup as EG:
            if self.audio_stream is not None:
                self.audio_stream.close()
            traceback.print_exception(EG)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--mode",
        type=str,
        default=DEFAULT_MODE,
        help="pixels to stream from",
        choices=["camera", "screen", "none"],
    )
    parser.add_argument(
        "--debug",
        action="store_true",
        help="log model responses to debug_log.txt",
    )
    args = parser.parse_args()
    main = AudioLoop(video_mode=args.mode, debug=args.debug)
    asyncio.run(main.run())


