import re

path = "/Users/z3robit/.gemini/antigravity/brain/0ab64b7b-40c0-46b5-8903-b63e2ff78e10/.system_generated/logs/transcript_full.jsonl"

with open(path, 'r', encoding='utf-8') as f:
    content = f.read()

# Find all view_file outputs
pattern = r"File Path: `file://([^`]+)`.*?Total Lines: \d+.*?The following code has been modified.*?<original_line>\. Please note.*?\\n(.*?)\\nThe above content shows the entire"
matches = re.finditer(pattern, content, re.DOTALL)

for m in matches:
    filepath = m.group(1)
    if "logic/fuzzy" in filepath:
        text = m.group(2)
        # unescape json newlines if it's inside a JSON string
        text = text.replace('\\n', '\n').replace('\\t', '\t').replace('\\"', '"')
        lines = text.split('\n')
        clean_lines = []
        for l in lines:
            lm = re.match(r"^\d+:\s(.*)", l)
            if lm:
                clean_lines.append(lm.group(1))
            elif re.match(r"^\d+:$", l):
                clean_lines.append("")
        
        with open(filepath, "w") as outf:
            outf.write("\n".join(clean_lines))
            print(f"Restored {filepath}")

