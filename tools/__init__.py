import importlib
import os
import glob

# Dictionary to store tool implementations: {name: function}
TOOL_MAP = {}
# List to store tool declarations for the API config
TOOL_DECLARATIONS = []

def load_tools():
    """Dynamically loads all tools from the current directory."""
    global TOOL_MAP, TOOL_DECLARATIONS
    
    TOOL_MAP.clear()
    TOOL_DECLARATIONS.clear()
    
    # Get all .py files in this directory except __init__.py
    current_dir = os.path.dirname(__file__)
    tool_files = glob.glob(os.path.join(current_dir, "*.py"))
    
    for file_path in tool_files:
        module_name = os.path.basename(file_path)[:-3]
        if module_name == "__init__":
            continue
            
        try:
            # Import the module dynamically
            module = importlib.import_module(f"tools.{module_name}")
            
            # Check if it has the expected attributes
            if hasattr(module, "declaration") and hasattr(module, module_name):
                declaration = getattr(module, "declaration")
                func = getattr(module, module_name)
                
                TOOL_MAP[module_name] = func
                TOOL_DECLARATIONS.append(declaration)
                # print(f"[Tool Loaded] {module_name}")
        except Exception as e:
            print(f"[Error loading tool {module_name}]: {e}")

# Initial load
load_tools()
