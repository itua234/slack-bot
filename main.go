package main

import (
	"fmt"
	"log"
	"os"

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
