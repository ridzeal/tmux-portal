package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins since behind Cloudflare
	},
}

type TmuxClient struct {
	conn    *websocket.Conn
	ptyFile *os.File
	cmd     *exec.Cmd
	closed  bool
	mu      sync.Mutex
}

func handleWebSocket(c *gin.Context) {
	sessionName := c.Query("session")
	if sessionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session parameter required"})
		return
	}

	// Get terminal size from query params (default 80x24)
	rows := c.Query("rows")
	cols := c.Query("cols")

	width, height := 80, 24
	if cols != "" {
		if _, err := fmt.Sscanf(cols, "%d", &width); err != nil {
			width = 80
		}
	}
	if rows != "" {
		if _, err := fmt.Sscanf(rows, "%d", &height); err != nil {
			height = 24
		}
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &TmuxClient{conn: conn, closed: false}

	// Start tmux with PTY
	if err := client.startTmux(sessionName, width, height); err != nil {
		log.Printf("Failed to start tmux: %v", err)
		conn.Close()
		return
	}

	// Handle connection
	client.handleConnection()
}

func (tc *TmuxClient) startTmux(sessionName string, width, height int) error {
	// Start tmux attach command with PTY
	tc.cmd = exec.Command("tmux", "-u", "-2", "attach-session", "-t", sessionName)

	// Set environment variables for proper terminal
	tc.cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"LANG=en_US.UTF-8",
	)

	// Start PTY with winsize
	ptyFile, err := pty.StartWithSize(tc.cmd, &pty.Winsize{
		Rows: uint16(height),
		Cols: uint16(width),
	})
	if err != nil {
		return err
	}

	tc.ptyFile = ptyFile
	log.Printf("Started PTY for session: %s (size: %dx%d)", sessionName, width, height)

	return nil
}

func (tc *TmuxClient) handleConnection() {
	defer tc.close()

	// Start goroutines for bidirectional communication
	var wg sync.WaitGroup
	wg.Add(2)

	// Read from PTY and send to WebSocket
	go func() {
		defer wg.Done()
		tc.readFromPTY()
	}()

	// Read from WebSocket and send to PTY
	go func() {
		defer wg.Done()
		tc.readFromWebSocket()
	}()

	wg.Wait()
}

func (tc *TmuxClient) readFromPTY() {
	buf := make([]byte, 1024)

	for {
		n, err := tc.ptyFile.Read(buf)
		if err != nil {
			if err != io.EOF {
				tc.mu.Lock()
				if !tc.closed {
					log.Printf("PTY read error: %v", err)
				}
				tc.mu.Unlock()
			}
			return
		}

		// Encode to base64 string to avoid encoding issues
		encoded := base64.StdEncoding.EncodeToString(buf[:n])

		tc.mu.Lock()
		if !tc.closed {
			if err := tc.conn.WriteMessage(websocket.TextMessage, []byte(encoded)); err != nil {
				log.Printf("WebSocket write error: %v", err)
				tc.close()
			}
		}
		tc.mu.Unlock()
	}
}

func (tc *TmuxClient) readFromWebSocket() {
	for {
		_, message, err := tc.conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		// Decode base64 message
		data, err := base64.StdEncoding.DecodeString(string(message))
		if err != nil {
			log.Printf("Base64 decode error: %v", err)
			continue
		}

		// Write to PTY
		tc.mu.Lock()
		if !tc.closed {
			if _, err := tc.ptyFile.Write(data); err != nil {
				log.Printf("PTY write error: %v", err)
				tc.close()
			}
		}
		tc.mu.Unlock()
	}
}

func (tc *TmuxClient) close() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	if tc.closed {
		return
	}

	tc.closed = true

	// Close connection
	tc.conn.Close()

	// Close PTY
	if tc.ptyFile != nil {
		tc.ptyFile.Close()
	}

	// Kill tmux process
	if tc.cmd != nil && tc.cmd.Process != nil {
		tc.cmd.Process.Kill()
		tc.cmd.Wait()
	}

	log.Println("Connection closed")
}
