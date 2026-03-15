import os
import config_utils

def read_user_memory():
    """Reads saved information about the user from the memory file."""
    try:
        if os.path.exists(config_utils.MEMORY_FILE):
            with open(config_utils.MEMORY_FILE, "r", encoding="utf-8") as f:
                content = f.read().strip()
                if content:
                    return {"status": "success", "content": content}
                return {"status": "success", "message": "Memory is empty"}
        return {"status": "success", "message": "Memory file not yet created"}
    except Exception as e:
        return {"status": "error", "message": str(e)}

declaration = {
    "name": "read_user_memory",
    "description": "Reads the user's saved history and information (plans, deadlines, preferences).",
    "parameters": {"type": "OBJECT", "properties": {}}
}
