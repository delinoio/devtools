# Delino DevTools

A collection of command-line tools to improve development productivity.

## Installation

```bash
# Install all tools
go install ./cmd/...

# Install a specific tool
go install ./cmd/syncai
```

## Tools

### syncai

Finds all `AGENTS.md` files in a Git repository and merges them into a single `CLAUDE.md` file. This allows hierarchical management of AI agent rules per directory.

#### Usage

```bash
# Basic execution - generates CLAUDE.md file
syncai

# Dry-run mode - preview without making changes
syncai --dry-run

# Verbose output mode
syncai -v

# Combine dry-run and verbose output
syncai --dry-run -v
```

#### How it works

1. Finds the Git repository root from the current directory
2. Recursively searches for all `AGENTS.md` files in the repository
3. Sorts the found files by directory path
4. Merges the content of each `AGENTS.md` file with directory information
5. Saves the result as `CLAUDE.md` in the Git root

#### Output format

The generated `CLAUDE.md` file has the following structure:

```markdown
# AI Agent Rules

When working in a specific directory, apply the rules from that directory and all parent directories up to the root.

## While working on `.`

*Source: `AGENTS.md`*

[Content of root AGENTS.md]

---

## While working on `sub/directory`

*Source: `sub/directory/AGENTS.md`*

[Content of that directory's AGENTS.md]
```

#### Options

- `--dry-run`: Preview the content that would be generated without actually creating the file
- `-v`: Verbose output showing processing details (Git root path, list of found files, etc.)

#### Examples

```bash
# Run in your project
cd /path/to/your/project
syncai
# ✓ Successfully merged 3 AGENTS.md file(s) into CLAUDE.md

# Preview
syncai --dry-run
# === DRY RUN MODE ===
# Would write to: /path/to/your/project/CLAUDE.md
#
# === Content Preview ===
# [Merged content displayed]
```