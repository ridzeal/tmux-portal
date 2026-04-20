package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Session represents a tmux session
type Session struct {
	ID      string
	Name    string
	Windows string
	Created string
}

// ListSessions returns all active tmux sessions
func ListSessions() ([]Session, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_id}:#{session_name}:#{session_windows}:#{session_created_string}")
	output, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 1 {
				// No sessions exist
				return []Session{}, nil
			}
		}
		return []Session{}, nil // Return empty array on error instead of nil
	}

	var sessions []Session
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 4 {
			sessions = append(sessions, Session{
				ID:      parts[0],
				Name:    parts[1],
				Windows: parts[2],
				Created: parts[3],
			})
		}
	}

	return sessions, nil
}

// CreateSession creates a new tmux session with the given name.
func CreateSession(name string) error {
	if !isValidSessionName(name) {
		return fmt.Errorf("invalid session name: %q (must match [a-zA-Z0-9._-])", name)
	}
	cmd := exec.Command("tmux", "new-session", "-d", "-s", name)
	return cmd.Run()
}

// KillSession kills the tmux session with the given name.
func KillSession(name string) error {
	if !isValidSessionName(name) {
		return fmt.Errorf("invalid session name: %q", name)
	}
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	return cmd.Run()
}

// GetSessionPath returns the socket path for a session
func GetSessionPath() string {
	// Check if TMUX env var is set (for when running inside tmux)
	if socket := os.Getenv("TMUX"); socket != "" {
		re := regexp.MustCompile(`^([^,]+)`)
		matches := re.FindStringSubmatch(socket)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	// Default tmux socket location
	return os.Getenv("HOME") + "/.tmux.sock"
}

var sessionNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// isValidSessionName checks that a session name matches tmux's allowed characters.
func isValidSessionName(name string) bool {
	return name != "" && sessionNameRe.MatchString(name)
}
