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

For detailed usage and examples, see [cmd/syncai/README.md](cmd/syncai/README.md).