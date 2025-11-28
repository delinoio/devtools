package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Show what would be done without making changes")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

	// Get git root directory
	gitRoot, err := getGitRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: not in a git repository: %v\n", err)
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Git root: %s\n", gitRoot)
	}

	// Find all AGENTS.md files
	agentFiles, err := findAgentFiles(gitRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding AGENTS.md files: %v\n", err)
		os.Exit(1)
	}

	if len(agentFiles) == 0 {
		fmt.Println("No AGENTS.md files found in repository")
		return
	}

	if *verbose {
		fmt.Printf("Found %d AGENTS.md file(s):\n", len(agentFiles))
		for _, f := range agentFiles {
			relPath, _ := filepath.Rel(gitRoot, f)
			fmt.Printf("  - %s\n", relPath)
		}
	}

	// Merge all AGENTS.md files
	mergedContent, err := mergeAgentFiles(gitRoot, agentFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error merging AGENTS.md files: %v\n", err)
		os.Exit(1)
	}

	claudePath := filepath.Join(gitRoot, "CLAUDE.md")

	if *dryRun {
		fmt.Println("=== DRY RUN MODE ===")
		fmt.Printf("Would write to: %s\n", claudePath)
		fmt.Println("\n=== Content Preview ===")
		fmt.Println(mergedContent)
		return
	}

	// Write merged content to CLAUDE.md
	err = os.WriteFile(claudePath, []byte(mergedContent), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing CLAUDE.md: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Successfully merged %d AGENTS.md file(s) into CLAUDE.md\n", len(agentFiles))
}

// getGitRoot returns the root directory of the git repository
func getGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// findAgentFiles finds all AGENTS.md files in the repository
func findAgentFiles(gitRoot string) ([]string, error) {
	var agentFiles []string

	err := filepath.Walk(gitRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Check if file is named AGENTS.md
		if !info.IsDir() && info.Name() == "AGENTS.md" {
			agentFiles = append(agentFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort files for consistent ordering
	sort.Strings(agentFiles)

	return agentFiles, nil
}

// mergeAgentFiles merges all AGENTS.md files into a single content
func mergeAgentFiles(gitRoot string, agentFiles []string) (string, error) {
	var buffer bytes.Buffer

	// Build a tree structure to show hierarchy
	buffer.WriteString("# AI Agent Rules\n\n")
	buffer.WriteString("When working in a specific directory, apply the rules from that directory and all parent directories up to the root.\n\n")

	// Merge each AGENTS.md file
	for i, agentFile := range agentFiles {
		// Get relative path for the section header
		relPath, err := filepath.Rel(gitRoot, agentFile)
		if err != nil {
			relPath = agentFile
		}

		// Read the file content
		content, err := os.ReadFile(agentFile)
		if err != nil {
			return "", fmt.Errorf("error reading %s: %v", agentFile, err)
		}

		// Get directory name for section title
		dirName := filepath.Dir(relPath)
		if dirName == "." {
			dirName = "."
		}

		// Write section header
		buffer.WriteString(fmt.Sprintf("## While working on `%s`\n\n", dirName))
		buffer.WriteString(fmt.Sprintf("*Source: `%s`*\n\n", relPath))

		// Write content
		buffer.Write(content)

		// Add separator between files (except for the last one)
		if i < len(agentFiles)-1 {
			buffer.WriteString("\n\n---\n\n")
		}
	}

	return buffer.String(), nil
}
