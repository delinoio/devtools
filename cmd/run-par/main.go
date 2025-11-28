package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Parse flags
	continueOnError := flag.Bool("continue-on-error", false, "Continue running other commands if one fails")
	flag.Parse()

	// Get command arguments
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("Usage: run-par [--continue-on-error] 'cmd1 arg1' 'cmd2 arg2' ...")
		os.Exit(1)
	}

	// Parse commands
	// Use sh -c to execute commands, which properly handles &&, ||, ;, pipes, etc.
	commands := make([]*Command, 0, len(args))
	for i, arg := range args {
		if arg == "" {
			continue
		}

		cmd := &Command{
			Index:       i,
			Name:        "sh",
			Args:        []string{"-c", arg},
			FullCommand: arg,
			Status:      StatusPending,
			Output:      make([]string, 0),
		}
		commands = append(commands, cmd)
	}

	if len(commands) == 0 {
		fmt.Println("No valid commands provided")
		os.Exit(1)
	}

	// Create update channel
	updates := make(chan CommandUpdate, 100)

	// Start commands in background
	ctx := context.Background()
	go func() {
		if err := RunCommands(ctx, commands, updates, *continueOnError); err != nil {
			// Error handling is done in the runner
		}
	}()

	// Create and run TUI
	m := newModel(commands, updates)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}

	// Print final status and logs
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("FINAL RESULTS")
	fmt.Println(strings.Repeat("=", 80))

	for _, cmd := range commands {
		cmd.mu.RLock()
		status := cmd.Status
		exitCode := cmd.ExitCode
		output := cmd.Output
		cmd.mu.RUnlock()

		// Print command header
		fmt.Println()
		statusStr := status.String()
		switch status {
		case StatusSuccess:
			fmt.Printf("✓ %s [%s]\n", cmd.FullCommand, statusStr)
		case StatusFailed:
			fmt.Printf("✗ %s [%s ] (exit code: %d)\n", cmd.FullCommand, statusStr, exitCode)
		default:
			fmt.Printf("○ %s [%s]\n", cmd.FullCommand, statusStr)
		}

		// Print separator
		fmt.Println(strings.Repeat("-", 80))

		// Print output
		if len(output) > 0 {
			for _, line := range output {
				fmt.Println(line)
			}
		} else {
			fmt.Println("(no output)")
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))

	// Exit with non-zero if any command failed
	for _, cmd := range commands {
		cmd.mu.RLock()
		status := cmd.Status
		cmd.mu.RUnlock()

		if status == StatusFailed {
			os.Exit(1)
		}
	}
}
