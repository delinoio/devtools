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
	commands := make([]*Command, 0, len(args))
	for i, arg := range args {
		parts := strings.Fields(arg)
		if len(parts) == 0 {
			continue
		}

		cmd := &Command{
			Index:       i,
			Name:        parts[0],
			Args:        parts[1:],
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

	// Print final status
	fmt.Println("\nFinal Status:")
	for _, cmd := range commands {
		cmd.mu.RLock()
		status := cmd.Status
		exitCode := cmd.ExitCode
		cmd.mu.RUnlock()

		statusStr := status.String()
		if status == StatusSuccess {
			fmt.Printf("✓ %s: %s\n", cmd.FullCommand, statusStr)
		} else if status == StatusFailed {
			fmt.Printf("✗ %s: %s (exit code: %d)\n", cmd.FullCommand, statusStr, exitCode)
		} else {
			fmt.Printf("○ %s: %s\n", cmd.FullCommand, statusStr)
		}
	}

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
