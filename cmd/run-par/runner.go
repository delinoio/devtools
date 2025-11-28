package main

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"sync"
)

type CommandStatus int

const (
	StatusPending CommandStatus = iota
	StatusRunning
	StatusSuccess
	StatusFailed
)

func (s CommandStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusSuccess:
		return "success"
	case StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

type Command struct {
	Index       int
	Name        string
	Args        []string
	FullCommand string
	Status      CommandStatus
	Output      []string
	ExitCode    int
	mu          sync.RWMutex
}

type UpdateType int

const (
	UpdateStatus UpdateType = iota
	UpdateOutput
	UpdateComplete
)

type CommandUpdate struct {
	Index    int
	Type     UpdateType
	Status   CommandStatus
	Line     string
	ExitCode int
}

func RunCommands(ctx context.Context, commands []*Command, updates chan<- CommandUpdate, continueOnError bool) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(commands))

	// Create a cancellable context for fail-fast behavior
	cmdCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, cmd := range commands {
		wg.Add(1)
		go func(c *Command) {
			defer wg.Done()
			if err := runSingleCommand(cmdCtx, c, updates); err != nil {
				if !continueOnError {
					errChan <- err
					cancel() // Cancel all other commands
				}
			}
		}(cmd)
	}

	// Wait for all commands to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Check for errors
	for err := range errChan {
		if err != nil && !continueOnError {
			return err
		}
	}

	return nil
}

func runSingleCommand(ctx context.Context, cmd *Command, updates chan<- CommandUpdate) error {
	// Check if context is already cancelled before starting
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Update status to running
	cmd.mu.Lock()
	cmd.Status = StatusRunning
	cmd.mu.Unlock()

	updates <- CommandUpdate{
		Index:  cmd.Index,
		Type:   UpdateStatus,
		Status: StatusRunning,
	}

	// Create command
	execCmd := exec.CommandContext(ctx, cmd.Name, cmd.Args...)

	// Get stdout and stderr pipes
	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := execCmd.StderrPipe()
	if err != nil {
		return err
	}

	// Start command
	if err := execCmd.Start(); err != nil {
		cmd.mu.Lock()
		cmd.Status = StatusFailed
		cmd.ExitCode = 1
		cmd.mu.Unlock()

		updates <- CommandUpdate{
			Index:    cmd.Index,
			Type:     UpdateComplete,
			Status:   StatusFailed,
			ExitCode: 1,
		}
		return err
	}

	// Read output
	var wg sync.WaitGroup
	wg.Add(2)

	go streamOutput(stdout, cmd, updates, &wg)
	go streamOutput(stderr, cmd, updates, &wg)

	wg.Wait()

	// Wait for command to finish
	err = execCmd.Wait()

	exitCode := 0
	status := StatusSuccess
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
		status = StatusFailed
	}

	cmd.mu.Lock()
	cmd.Status = status
	cmd.ExitCode = exitCode
	cmd.mu.Unlock()

	updates <- CommandUpdate{
		Index:    cmd.Index,
		Type:     UpdateComplete,
		Status:   status,
		ExitCode: exitCode,
	}

	return err
}

func streamOutput(reader io.Reader, cmd *Command, updates chan<- CommandUpdate, wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		cmd.mu.Lock()
		cmd.Output = append(cmd.Output, line)
		cmd.mu.Unlock()

		updates <- CommandUpdate{
			Index: cmd.Index,
			Type:  UpdateOutput,
			Line:  line,
		}
	}
}
