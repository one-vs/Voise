import os
import datetime
import config_utils

def save_user_memory(info: str):
    """Saves important information about the user to the memory file with a timestamp."""
    try:
        now = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        entry = f"[{now}] {info}"

        existing = ""
        if os.path.exists(config_utils.MEMORY_FILE):
            with open(config_utils.MEMORY_FILE, "r", encoding="utf-8") as f:
                existing = f.read()

        with open(config_utils.MEMORY_FILE, "a", encoding="utf-8") as f:
            if existing and not existing.endswith("\n"):
                f.write("\n")
            f.write(f"{entry}\n")
        print(f"\n[Memory saved: {entry}]")
        return {"status": "success", "message": "Information saved"}
    except Exception as e:
        return {"status": "error", "message": str(e)}

declaration = {
    "name": "save_user_memory",
    "description": "Saves important information about the user (tasks, plans, facts) to persistent memory.",
    "parameters": {
        "type": "OBJECT",
        "properties": {
            "info": {"type": "STRING", "description": "Information to save"}
        },
        "required": ["info"]
    }
}
