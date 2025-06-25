package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/koneksi/mcp-server/internal/koneksi"
	"github.com/koneksi/mcp-server/internal/mcp"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize Koneksi client
	clientID := os.Getenv("KONEKSI_API_CLIENT_ID")
	clientSecret := os.Getenv("KONEKSI_API_CLIENT_SECRET")
	baseURL := os.Getenv("KONEKSI_API_BASE_URL")

	if baseURL == "" {
		baseURL = "https://staging.koneksi.co.kr"
	}

	if clientID == "" || clientSecret == "" {
		log.Fatal("KONEKSI_API_CLIENT_ID and KONEKSI_API_CLIENT_SECRET must be set")
	}

	directoryID := os.Getenv("KONEKSI_DIRECTORY_ID")
	koneksiClient := koneksi.NewClient(baseURL, clientID, clientSecret, directoryID)

	// Create MCP server
	server := mcp.NewServer("koneksi-storage", "1.0.0", koneksiClient)

	// Setup stdin/stdout communication
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	log.Println("Koneksi MCP server started")

	// Main message loop
	for scanner.Scan() {
		request := scanner.Text()
		
		// Parse request to check if it's a notification
		var req map[string]interface{}
		if err := json.Unmarshal([]byte(request), &req); err != nil {
			log.Printf("Error parsing request: %v", err)
			continue
		}
		
		response, err := server.HandleRequest(request)
		if err != nil {
			log.Printf("Error handling request: %v", err)
			
			// For notifications (no ID), don't send any response
			if req["id"] == nil {
				continue
			}
			
			errorResponse := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req["id"],
				"error": map[string]interface{}{
					"code":    -32603,
					"message": err.Error(),
				},
			}
			
			encoder.Encode(errorResponse)
			continue
		}

		// Don't send response for notifications
		if req["id"] == nil {
			continue
		}

		if err := encoder.Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading stdin: %v", err)
	}
}
