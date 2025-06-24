package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
)

// Global Slack client instance
var slackClient *slack.Client
var slackSigningSecret string

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	slackBotToken := os.Getenv("SLACK_BOT_TOKEN")
	slackSigningSecret = os.Getenv("SLACK_SIGNING_SECRET")
	if slackBotToken == "" || slackSigningSecret == "" {
		log.Fatal("SLACK_BOT_TOKEN and SLACK_SIGNING_SECRET must be set in .env")
	}
	// Initialize Slack client
	slackClient = slack.New(slackBotToken)
	fmt.Println(slackClient)

	router := gin.Default()

	// Use a custom middleware for Slack request verification
	router.Use(verifySlackRequestMiddleware)

	// Slack Events API endpoint
	router.POST("/slack/events", handleSlackEvents)

	// Start the Gin server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// verifySlackRequestMiddleware verifies incoming requests from Slack
func verifySlackRequestMiddleware(c *gin.Context) {
	// Read the raw request body
	body, err := io.ReadAll((c.Request.Body))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		c.Abort()
		return
	}
	// Restore the body for subsequent handlers
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// Get Slack headers
	timestamp := c.GetHeader("X-Slack-Request-Timestamp")
	//signature := c.GetHeader("X-Slack-Signature")
	// Verify the request
	verifier, err := slack.NewSecretsVerifier(c.Request.Header, slackSigningSecret)
	if err != nil {
		log.Printf("Error creating verifier: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		c.Abort()
		return
	}

	// Write the raw body to the verifier
	_, err = verifier.Write(body)
	if err != nil {
		log.Printf("Error writing body to verifier: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		c.Abort()
		return
	}

	if err = verifier.Ensure(); err != nil {
		log.Printf("Slack signature verification failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Slack signature verification failed"})
		c.Abort()
		return
	}

	// Check for replay attacks (timestamp within 5 minutes)
	t, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		log.Printf("Invalid timestamp: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timestamp"})
		c.Abort()
		return
	}
	if time.Since(time.Unix(t, 0)) > 5*time.Minute {
		log.Print("Request timestamp too old (replay attack potential)")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Request timestamp too old"})
		c.Abort()
		return
	}

	c.Next()
}

func handleSlackEvents(c *gin.Context) {

}
