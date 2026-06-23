import json
import re

transcript_path = "/Users/z3robit/.gemini/antigravity/brain/0ab64b7b-40c0-46b5-8903-b63e2ff78e10/.system_generated/logs/transcript_full.jsonl"

file_contents = {}

with open(transcript_path, 'r') as f:
    for line in f:
        try:
            data = json.loads(line)
        except:
            continue
        if data.get("type") == "TOOL_RESPONSE":
            for resp in data.get("tool_responses", []):
                out = resp.get("response", {}).get("output", "")
                if out:
                    m = re.search(r"File Path: `file://([^`]+)`", out)
                    if m:
                        filepath = m.group(1)
                        if "logic/fuzzy/" in filepath:
                            parts = out.split("The following code has been modified to include a line number before every line, in the format: <line_number>: <original_line>. Please note that any changes targeting the original code should remove the line number, colon, and leading space.\n")
                            if len(parts) == 2:
                                text = parts[1]
                                text = text.split("\nThe above content")[0]
                                clean_lines = []
                                for l in text.split("\n"):
                                    lm = re.match(r"^\d+:\s(.*)", l)
                                    if lm:
                                        clean_lines.append(lm.group(1))
                                    else:
                                        if re.match(r"^\d+:$", l):
                                            clean_lines.append("")
                                        else:
                                            clean_lines.append(l)
                                file_contents[filepath] = "\n".join(clean_lines)

for fp, content in file_contents.items():
    print(f"Restoring {fp}")
    with open(fp, "w") as f:
        f.write(content)

print("Done.")
