package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

type MCPBridge struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	scanner   *bufio.Scanner
	mu        sync.Mutex
	requestID int
	pending   map[interface{}]chan json.RawMessage
}

type MCPRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params"`
}

type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type APIRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

type APIResponse struct {
	Success bool            `json:"success"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   string          `json:"error,omitempty"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for demo purposes
	},
}

func NewMCPBridge() (*MCPBridge, error) {
	// Start the MCP server as a subprocess
	cmd := exec.Command("go", "run", "./cmd/koneksi-mcp-server")
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MCP server: %w", err)
	}
	
	bridge := &MCPBridge{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		scanner: bufio.NewScanner(stdout),
		pending: make(map[interface{}]chan json.RawMessage),
	}
	
	// Start reading responses
	go bridge.readResponses()
	go bridge.readErrors()
	
	// Initialize the MCP connection
	if err := bridge.initialize(); err != nil {
		bridge.Close()
		return nil, fmt.Errorf("failed to initialize MCP: %w", err)
	}
	
	return bridge, nil
}

func (b *MCPBridge) initialize() error {
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      "init",
		Method:  "initialize",
		Params:  map[string]interface{}{},
	}
	
	resp, err := b.sendRequest(req)
	if err != nil {
		return err
	}
	
	log.Printf("MCP initialized: %s", resp)
	return nil
}

func (b *MCPBridge) sendRequest(req MCPRequest) (json.RawMessage, error) {
	b.mu.Lock()
	if req.ID == nil {
		b.requestID++
		req.ID = b.requestID
	}
	
	// Create response channel
	respChan := make(chan json.RawMessage, 1)
	b.pending[req.ID] = respChan
	b.mu.Unlock()
	
	// Send request
	data, err := json.Marshal(req)
	if err != nil {
		b.mu.Lock()
		delete(b.pending, req.ID)
		b.mu.Unlock()
		return nil, err
	}
	
	if _, err := b.stdin.Write(append(data, '\n')); err != nil {
		b.mu.Lock()
		delete(b.pending, req.ID)
		b.mu.Unlock()
		return nil, err
	}
	
	// Wait for response with timeout
	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(30 * time.Second):
		b.mu.Lock()
		delete(b.pending, req.ID)
		b.mu.Unlock()
		return nil, fmt.Errorf("request timeout")
	}
}

func (b *MCPBridge) readResponses() {
	for b.scanner.Scan() {
		line := b.scanner.Text()
		
		var resp MCPResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			log.Printf("Failed to parse MCP response: %v", err)
			continue
		}
		
		b.mu.Lock()
		if ch, ok := b.pending[resp.ID]; ok {
			delete(b.pending, resp.ID)
			if resp.Error != nil {
				log.Printf("MCP error: %v", resp.Error)
				ch <- nil
			} else {
				ch <- resp.Result
			}
		}
		b.mu.Unlock()
	}
}

func (b *MCPBridge) readErrors() {
	scanner := bufio.NewScanner(b.stderr)
	for scanner.Scan() {
		log.Printf("MCP stderr: %s", scanner.Text())
	}
}

func (b *MCPBridge) Close() error {
	b.stdin.Close()
	b.stdout.Close()
	b.stderr.Close()
	return b.cmd.Process.Kill()
}

func (b *MCPBridge) handleAPIRequest(w http.ResponseWriter, r *http.Request) {
	var apiReq APIRequest
	if err := json.NewDecoder(r.Body).Decode(&apiReq); err != nil {
		respondWithJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}
	
	// Convert API request to MCP request
	mcpReq := MCPRequest{
		JSONRPC: "2.0",
		Method:  apiReq.Method,
		Params:  apiReq.Params,
	}
	
	// Send to MCP server
	result, err := b.sendRequest(mcpReq)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	
	respondWithJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Result:  result,
	})
}

func (b *MCPBridge) handleToolCall(w http.ResponseWriter, r *http.Request) {
	var toolCall struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&toolCall); err != nil {
		respondWithJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}
	
	// Marshal arguments to JSON string
	argsJSON, err := json.Marshal(toolCall.Arguments)
	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Invalid arguments",
		})
		return
	}
	
	// Create MCP tool call request
	mcpReq := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      toolCall.Name,
			"arguments": string(argsJSON),
		},
	}
	
	// Send to MCP server
	result, err := b.sendRequest(mcpReq)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	
	respondWithJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Result:  result,
	})
}

func (b *MCPBridge) handleListTools(w http.ResponseWriter, r *http.Request) {
	mcpReq := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/list",
		Params:  map[string]interface{}{},
	}
	
	result, err := b.sendRequest(mcpReq)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}
	
	respondWithJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Result:  result,
	})
}

func (b *MCPBridge) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	
	for {
		var apiReq APIRequest
		if err := conn.ReadJSON(&apiReq); err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		
		// Convert to MCP request
		mcpReq := MCPRequest{
			JSONRPC: "2.0",
			Method:  apiReq.Method,
			Params:  apiReq.Params,
		}
		
		// Send to MCP server
		result, err := b.sendRequest(mcpReq)
		
		// Send response back via WebSocket
		resp := APIResponse{
			Success: err == nil,
		}
		if err != nil {
			resp.Error = err.Error()
		} else {
			resp.Result = result
		}
		
		if err := conn.WriteJSON(resp); err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}
}

func (b *MCPBridge) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (32MB max)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Failed to parse form",
		})
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "File is required",
		})
		return
	}
	defer file.Close()

	// Create temporary file
	tempFile, err := os.CreateTemp("", "upload-*.tmp")
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to create temporary file",
		})
		return
	}
	defer os.Remove(tempFile.Name())

	// Copy file content
	_, err = io.Copy(tempFile, file)
	if err != nil {
		tempFile.Close()
		respondWithJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to save file",
		})
		return
	}
	tempFile.Close()

	// Get directory ID from form
	directoryID := r.FormValue("directory_id")

	// Prepare arguments for MCP tool call
	args := map[string]interface{}{
		"filePath": tempFile.Name(),
	}
	if directoryID != "" {
		args["directoryId"] = directoryID
	}

	// Call MCP upload tool
	mcpReq := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      "upload_file",
			"arguments": string(mustMarshalJSON(args)),
		},
	}

	// Send to MCP server
	result, err := b.sendRequest(mcpReq)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Add filename to response
	response := map[string]interface{}{
		"success":  true,
		"result":   result,
		"filename": header.Filename,
		"size":     header.Size,
	}

	respondWithJSON(w, http.StatusOK, response)
}

func mustMarshalJSON(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
	
	// Create MCP bridge
	bridge, err := NewMCPBridge()
	if err != nil {
		log.Fatal(err)
	}
	defer bridge.Close()
	
	// Setup routes
	router := mux.NewRouter()
	
	// MCP endpoints
	router.HandleFunc("/api/v1/mcp/request", bridge.handleAPIRequest).Methods("POST")
	router.HandleFunc("/api/v1/mcp/tools/list", bridge.handleListTools).Methods("GET")
	router.HandleFunc("/api/v1/mcp/tools/call", bridge.handleToolCall).Methods("POST")
	
	// File upload endpoint
	router.HandleFunc("/api/v1/upload", bridge.handleFileUpload).Methods("POST")
	
	// WebSocket for real-time communication
	router.HandleFunc("/ws", bridge.handleWebSocket)
	
	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		respondWithJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
	}).Methods("GET")
	
	// Serve static files
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./cmd/koneksi-mcp-bridge/static/")))
	
	// CORS middleware
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})
	
	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	
	log.Printf("Starting MCP Bridge API on port %s", port)
	log.Println("Endpoints:")
	log.Println("  POST /api/v1/mcp/request - Send raw MCP request")
	log.Println("  GET  /api/v1/mcp/tools/list - List available tools")
	log.Println("  POST /api/v1/mcp/tools/call - Call a specific tool")
	log.Println("  WS   /ws - WebSocket connection for real-time communication")
	
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal(err)
	}
}