package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	// Clients connected to session updates
	sessionUpdateClients = make(map[*websocket.Conn]bool)
	sessionUpdateMutex   sync.RWMutex
)

// SessionUpdateMessage represents a session list update
type SessionUpdateMessage struct {
	Type     string    `json:"type"`     // "update", "create", "delete"
	Sessions []Session `json:"sessions"` // Full session list or changed session
	Name     string    `json:"name"`     // Session name (for create/delete)
}

// handleSessionUpdates handles WebSocket connections for session list updates
func handleSessionUpdates(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Session update WebSocket upgrade failed: %v", err)
		return
	}

	// Register client
	sessionUpdateMutex.Lock()
	sessionUpdateClients[conn] = true
	sessionUpdateMutex.Unlock()

	// Send initial session list
	sessions, _ := ListSessions()
	updateMsg := SessionUpdateMessage{
		Type:     "update",
		Sessions: sessions,
	}
	jsonData, _ := json.Marshal(updateMsg)
	conn.WriteMessage(websocket.TextMessage, jsonData)

	log.Println("Client subscribed to session updates")

	// Keep connection alive and handle disconnects
	defer func() {
		sessionUpdateMutex.Lock()
		delete(sessionUpdateClients, conn)
		sessionUpdateMutex.Unlock()
		conn.Close()
		log.Println("Client unsubscribed from session updates")
	}()

	// Send periodic pings to keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// broadcastSessionUpdate sends session updates to all connected clients
func broadcastSessionUpdate(msgType string, sessions []Session, sessionName string) {
	updateMsg := SessionUpdateMessage{
		Type:     msgType,
		Sessions: sessions,
		Name:     sessionName,
	}

	jsonData, err := json.Marshal(updateMsg)
	if err != nil {
		log.Printf("Failed to marshal session update: %v", err)
		return
	}

	sessionUpdateMutex.RLock()
	defer sessionUpdateMutex.RUnlock()

	// Send to all connected clients
	for conn := range sessionUpdateClients {
		if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
			log.Printf("Failed to send session update to client: %v", err)
			delete(sessionUpdateClients, conn)
			conn.Close()
		}
	}
}

// startSessionMonitor starts a goroutine to monitor tmux sessions and push updates
func startSessionMonitor() {
	go func() {
		lastSessions := make(map[string]bool)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		// Initial check
		sessions, _ := ListSessions()
		for _, session := range sessions {
			lastSessions[session.Name] = true
		}

		for range ticker.C {
			currentSessions, err := ListSessions()
			if err != nil {
				continue
			}

			// Build current session map
			currentSessionMap := make(map[string]bool)
			for _, session := range currentSessions {
				currentSessionMap[session.Name] = true
			}

			// Check for changes
			changed := false

			// Check for new sessions
			for _, session := range currentSessions {
				if !lastSessions[session.Name] {
					log.Printf("Session created: %s", session.Name)
					broadcastSessionUpdate("update", currentSessions, session.Name)
					changed = true
					break
				}
			}

			// Check for deleted sessions
			for sessionName := range lastSessions {
				if !currentSessionMap[sessionName] {
					log.Printf("Session deleted: %s", sessionName)
					broadcastSessionUpdate("update", currentSessions, sessionName)
					changed = true
					break
				}
			}

			// Update last known sessions
			if changed || len(lastSessions) != len(currentSessionMap) {
				lastSessions = currentSessionMap
			}
		}
	}()
}
