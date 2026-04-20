package main

import (
	"flag"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	port := flag.String("port", "7777", "Server port")
	flag.Parse()

	// Start session monitor
	startSessionMonitor()

	// Create Gin router
	router := gin.Default()

	// Serve static files
	router.Static("/static", "./static")

	// Routes
	router.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	router.GET("/api/sessions", listSessionsHandler)
	router.POST("/api/sessions", createSessionHandler)
	router.DELETE("/api/sessions/:name", killSessionHandler)
	router.GET("/ws", handleWebSocket)
	router.GET("/ws/sessions", handleSessionUpdates)

	// Start server
	log.Printf("Starting tmux-portal on port %s", *port)
	if err := router.Run(":" + *port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
