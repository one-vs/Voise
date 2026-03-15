import os
import datetime
import re
import config_utils

def update_user_memory(old_text: str, new_text: str):
    """Updates or deletes an existing entry in the user's memory.
    To delete an entry, pass an empty string or the word 'delete' in new_text.
    """
    try:
        if not os.path.exists(config_utils.MEMORY_FILE):
            return {"status": "error", "message": "Memory file not found"}

        now = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")

        with open(config_utils.MEMORY_FILE, "r", encoding="utf-8") as f:
            lines = f.readlines()

        found = False
        new_lines = []
        for line in lines:
            content_part = line.split("] ", 1)[-1].strip() if "] " in line else line.strip()

            if old_text in content_part and not found:
                found = True
                # Delete if new_text is empty or a delete command
                if not new_text or new_text.strip().lower() in ["delete", "remove", "[deleted]"]:
                    print(f"\n[Memory entry deleted: {old_text}]")
                    continue  # Skip the line = delete it

                timestamp_match = re.match(r"(\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\])", line)
                orig_timestamp = timestamp_match.group(1) if timestamp_match else f"[{now}]"

                updated_line = f'{orig_timestamp} "{new_text}" (updated: {now})\n'
                new_lines.append(updated_line)
                print(f"\n[Memory entry updated: {new_text}]")
            else:
                new_lines.append(line)

        if found:
            with open(config_utils.MEMORY_FILE, "w", encoding="utf-8") as f:
                f.writelines(new_lines)
            return {"status": "success", "message": "Entry successfully updated or deleted"}
        else:
            return {"status": "error", "message": f"Entry containing '{old_text}' not found"}

    except Exception as e:
        return {"status": "error", "message": str(e)}

declaration = {
    "name": "update_user_memory",
    "description": "Updates or deletes an existing memory entry. Use this when information has changed.",
    "parameters": {
        "type": "OBJECT",
        "properties": {
            "old_text": {"type": "STRING", "description": "Text or part of the old entry to replace or delete"},
            "new_text": {"type": "STRING", "description": "New text. Pass an empty string or 'delete' to remove the entry."}
        },
        "required": ["old_text", "new_text"]
    }
}
