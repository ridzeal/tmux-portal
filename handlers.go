package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func listSessionsHandler(c *gin.Context) {
	sessions, err := ListSessions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sessions)
}

func createSessionHandler(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	if err := CreateSession(req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast session update to all clients
	sessions, _ := ListSessions()
	broadcastSessionUpdate("update", sessions, req.Name)

	c.JSON(http.StatusCreated, gin.H{"message": "Session created", "name": req.Name})
}

func killSessionHandler(c *gin.Context) {
	name := c.Param("name")

	// Try to kill - ignore error if session already gone
	KillSession(name)

	// Broadcast session update to all clients
	sessions, _ := ListSessions()
	broadcastSessionUpdate("update", sessions, name)

	c.JSON(http.StatusOK, gin.H{"message": "Session killed", "name": name})
}
