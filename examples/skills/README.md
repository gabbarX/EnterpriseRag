# Skills Examples

This directory contains examples of the Agent Skills feature.

## Directory structure

```
skills/
├── README.md              # This file
└── pdf-processing/        # Example PDF processing skill
    ├── SKILL.md           # Main file (Level 2)
    ├── FORMS.md           # Supplementary document (Level 3)
    └── scripts/           # Executable scripts
        ├── analyze_form.py
        └── extract_text.py
```

## Quick start

### Run the demo

```bash
go run ./cmd/skills-demo/main.go
```

### Create a new skill

1. Create a new folder in this directory:

```bash
mkdir my-new-skill
```

2. Create `SKILL.md`:

```markdown
---
name: my-new-skill
description: Description of what this skill does and when to use it.
---

# My New Skill

Instructions for the agent...
```

3. Add scripts (optional):

```bash
mkdir my-new-skill/scripts
# Add your scripts here
```

## Example: pdf-processing

This is a fully working example skill. It demonstrates:

- **SKILL.md**: the main file, including its YAML frontmatter
- **FORMS.md**: a supplementary reference document
- **scripts/**: Python scripts that can be executed inside the sandbox

### Skill description

```yaml
name: pdf-processing
description: Extract text and tables from PDF files, fill forms, merge documents.
```

### Scripts included

| Script | Function |
|------|------|
| `analyze_form.py` | Analyses the form fields in a PDF |
| `extract_text.py` | Extracts text from a PDF |

### Usage example

The agent invokes the skill automatically, based on the user's request:

```
User: "Analyse this PDF form and tell me what fields it has"

Agent:
  1. Matches the request to the pdf-processing skill
  2. Calls read_file(path="skill://pdf-processing/SKILL.md") to load the skill
  3. Calls shell_exec(skill_name="pdf-processing", command=...) to run analyze_form.py
  4. Returns the analysis of the form fields
```
