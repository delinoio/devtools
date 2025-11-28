# run-par

Runs multiple shell commands in parallel with a terminal user interface (TUI) that displays real-time output and status for each command.

## Usage

```bash
# Run multiple commands in parallel
run-par 'npm test' 'npm run lint' 'npm run build'

# Continue running other commands even if one fails
run-par --continue-on-error 'npm test' 'npm run lint' 'npm run build'

# Run complex commands with pipes, redirects, and operators
run-par 'echo "test" | grep t' 'ls -la && pwd' 'sleep 2; echo done'
```

## How it works

1. Parses command-line arguments as shell commands (each argument is a separate command)
2. Executes each command using `sh -c`, which properly handles shell operators like `&&`, `||`, `;`, pipes, etc.
3. Runs all commands in parallel as separate goroutines
4. Displays a real-time TUI with:
   - Left sidebar: List of commands with status indicators
   - Right panel: Output logs for the selected command
   - Status bar: Navigation instructions or search query
5. Auto-exits when all commands complete (or when the first one fails in non-continue mode)
6. Prints a final summary with all command outputs and exit codes

## TUI Features

### Navigation
- **↑/k**: Move selection up
- **↓/j**: Move selection down
- **/**: Enter search mode to filter output
- **q/Ctrl+C**: Quit (commands will be terminated)

### Status Indicators
- **○**: Pending (not started yet)
- **◐**: Running (currently executing)
- **●**: Success (completed with exit code 0)
- **✗**: Failed (non-zero exit code)

### Search Mode
When you press `/`, you can type to filter the output of the selected command. Only lines matching your search query will be displayed. Press `Enter` to exit search mode while keeping the filter, or `Esc` to clear the search.

## Options

- `--continue-on-error`: By default, if any command fails, all other commands are cancelled. With this flag, all commands will run to completion regardless of failures.

## Examples

```bash
# Run tests, linting, and build in parallel
run-par 'npm test' 'npm run lint' 'npm run build'

# Run different language checks
run-par 'go test ./...' 'cargo test' 'npm test'

# Continue all commands even if some fail
run-par --continue-on-error 'exit 1' 'echo "still running"' 'sleep 2 && echo "done"'

# Complex shell commands with operators
run-par \
  'find . -name "*.go" | wc -l' \
  'git status && git log -1' \
  'docker ps || echo "Docker not running"'
```

## Exit Code

- Returns 0 if all commands succeed
- Returns 1 if any command fails
- In `--continue-on-error` mode, returns 1 if any command failed, even if others succeeded

## Final Output

After the TUI exits, run-par prints a comprehensive final report showing:
- Status of each command (✓ for success, ✗ for failure)
- Exit codes for failed commands
- Complete output logs for all commands
