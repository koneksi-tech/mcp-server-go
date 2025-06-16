# MCP Server Development Guide

## Table of Contents
1. [What is MCP?](#what-is-mcp)
2. [Architecture Overview](#architecture-overview)
3. [Building an MCP Server](#building-an-mcp-server)
4. [Protocol Details](#protocol-details)
5. [Creating Tools](#creating-tools)
6. [Testing Your Server](#testing-your-server)
7. [Integration Options](#integration-options)
8. [Best Practices](#best-practices)

## What is MCP?

MCP (Model Context Protocol) is a protocol that enables AI assistants like Claude to interact with external systems through a standardized interface. It uses JSON-RPC 2.0 over stdin/stdout for communication.

### Key Features:
- **Tool-based architecture**: Define specific actions the AI can perform
- **Bidirectional communication**: Request/response pattern
- **Transport agnostic**: Works over stdin/stdout, WebSocket, or HTTP
- **Language agnostic**: Implement in any programming language

## Architecture Overview

```
┌─────────────┐     JSON-RPC 2.0     ┌─────────────┐
│ AI Assistant│ ←─────────────────→  │ MCP Server  │
│  (Claude)   │    stdin/stdout      │             │
└─────────────┘                      └─────────────┘
                                            │
                                            ↓
                                     ┌─────────────┐
                                     │  External   │
                                     │   System    │
                                     └─────────────┘
```

## Building an MCP Server

### 1. Basic Server Structure (Go Example)

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
)

type MCPServer struct {
    name    string
    version string
}

type Request struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      interface{}     `json:"id"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params"`
}

type Response struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id"`
    Result  interface{} `json:"result,omitempty"`
    Error   *Error      `json:"error,omitempty"`
}

type Error struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

func main() {
    server := &MCPServer{
        name:    "my-mcp-server",
        version: "1.0.0",
    }
    
    scanner := bufio.NewScanner(os.Stdin)
    encoder := json.NewEncoder(os.Stdout)
    
    for scanner.Scan() {
        var req Request
        if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
            continue
        }
        
        response := server.handleRequest(req)
        encoder.Encode(response)
    }
}
```

### 2. Implement Required Methods

Every MCP server must implement these core methods:

#### Initialize Method
```go
func (s *MCPServer) handleInitialize(req Request) Response {
    return Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result: map[string]interface{}{
            "protocolVersion": "2024-11-05",
            "capabilities": map[string]interface{}{
                "tools": map[string]interface{}{},
            },
            "serverInfo": map[string]interface{}{
                "name":    s.name,
                "version": s.version,
            },
        },
    }
}
```

#### Tools List Method
```go
func (s *MCPServer) handleToolsList(req Request) Response {
    tools := []map[string]interface{}{
        {
            "name":        "read_file",
            "description": "Read contents of a file",
            "inputSchema": map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "path": map[string]interface{}{
                        "type":        "string",
                        "description": "File path to read",
                    },
                },
                "required": []string{"path"},
            },
        },
    }
    
    return Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result: map[string]interface{}{
            "tools": tools,
        },
    }
}
```

#### Tools Call Method
```go
func (s *MCPServer) handleToolCall(req Request) Response {
    var params struct {
        Name      string `json:"name"`
        Arguments string `json:"arguments"`
    }
    json.Unmarshal(req.Params, &params)
    
    // Parse tool arguments
    var args map[string]interface{}
    json.Unmarshal([]byte(params.Arguments), &args)
    
    // Execute tool based on name
    switch params.Name {
    case "read_file":
        result := s.readFile(args)
        return Response{
            JSONRPC: "2.0",
            ID:      req.ID,
            Result:  result,
        }
    default:
        return Response{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error: &Error{
                Code:    -32601,
                Message: "Tool not found",
            },
        }
    }
}
```

### 3. Request Router
```go
func (s *MCPServer) handleRequest(req Request) Response {
    switch req.Method {
    case "initialize":
        return s.handleInitialize(req)
    case "tools/list":
        return s.handleToolsList(req)
    case "tools/call":
        return s.handleToolCall(req)
    default:
        return Response{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error: &Error{
                Code:    -32601,
                Message: "Method not found",
            },
        }
    }
}
```

## Protocol Details

### Request Format
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
        "name": "read_file",
        "arguments": "{\"path\": \"/tmp/test.txt\"}"
    }
}
```

### Response Format
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "content": [{
            "type": "text",
            "text": "File contents here..."
        }]
    }
}
```

### Error Response
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "error": {
        "code": -32603,
        "message": "Internal error: File not found"
    }
}
```

## Creating Tools

### Tool Definition Structure
```go
type Tool struct {
    Name        string      `json:"name"`
    Description string      `json:"description"`
    InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
    Type       string              `json:"type"`
    Properties map[string]Property `json:"properties"`
    Required   []string            `json:"required"`
}

type Property struct {
    Type        string `json:"type"`
    Description string `json:"description"`
    // Optional fields
    Enum    []string `json:"enum,omitempty"`
    Default string   `json:"default,omitempty"`
}
```

### Example Tools

#### File Operations Tool
```go
{
    "name": "write_file",
    "description": "Write content to a file",
    "inputSchema": {
        "type": "object",
        "properties": {
            "path": {
                "type": "string",
                "description": "File path to write"
            },
            "content": {
                "type": "string",
                "description": "Content to write"
            },
            "append": {
                "type": "boolean",
                "description": "Append to file instead of overwriting",
                "default": "false"
            }
        },
        "required": ["path", "content"]
    }
}
```

#### API Request Tool
```go
{
    "name": "http_request",
    "description": "Make an HTTP request",
    "inputSchema": {
        "type": "object",
        "properties": {
            "url": {
                "type": "string",
                "description": "URL to request"
            },
            "method": {
                "type": "string",
                "description": "HTTP method",
                "enum": ["GET", "POST", "PUT", "DELETE"],
                "default": "GET"
            },
            "headers": {
                "type": "object",
                "description": "HTTP headers"
            },
            "body": {
                "type": "string",
                "description": "Request body"
            }
        },
        "required": ["url"]
    }
}
```

## Testing Your Server

### 1. Manual Testing Script
```bash
#!/bin/bash
# test_mcp.sh

# Start your MCP server
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./your-mcp-server

# List available tools
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | ./your-mcp-server

# Call a tool
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"read_file","arguments":"{\"path\":\"/tmp/test.txt\"}"}}' | ./your-mcp-server
```

### 2. Automated Test Suite (Go)
```go
func TestMCPServer(t *testing.T) {
    // Create pipes for stdin/stdout
    stdin, stdinWriter := io.Pipe()
    stdoutReader, stdout := io.Pipe()
    
    // Start server with custom stdin/stdout
    go runServer(stdin, stdout)
    
    // Send initialize request
    request := Request{
        JSONRPC: "2.0",
        ID:      1,
        Method:  "initialize",
        Params:  json.RawMessage("{}"),
    }
    
    json.NewEncoder(stdinWriter).Encode(request)
    
    // Read response
    var response Response
    json.NewDecoder(stdoutReader).Decode(&response)
    
    // Verify response
    if response.Error != nil {
        t.Fatalf("Initialize failed: %v", response.Error)
    }
}
```

### 3. Integration Testing
```python
# Python test client
import json
import subprocess

def test_mcp_server():
    # Start MCP server as subprocess
    proc = subprocess.Popen(
        ['./your-mcp-server'],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        text=True
    )
    
    # Send request
    request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/list",
        "params": {}
    }
    
    proc.stdin.write(json.dumps(request) + '\n')
    proc.stdin.flush()
    
    # Read response
    response = json.loads(proc.stdout.readline())
    print(f"Tools: {response['result']['tools']}")
```

## Integration Options

### 1. Direct Integration with Claude
```json
{
    "mcpServers": {
        "my-server": {
            "command": "node",
            "args": ["./my-mcp-server.js"]
        }
    }
}
```

### 2. REST API Bridge
Create a REST API wrapper (like we did with the bridge server):

```go
// Convert REST to MCP
func handleRESTRequest(w http.ResponseWriter, r *http.Request) {
    // Parse REST request
    var restReq map[string]interface{}
    json.NewDecoder(r.Body).Decode(&restReq)
    
    // Convert to MCP request
    mcpReq := Request{
        JSONRPC: "2.0",
        ID:      generateID(),
        Method:  "tools/call",
        Params:  convertToMCPParams(restReq),
    }
    
    // Send to MCP server
    response := mcpClient.Send(mcpReq)
    
    // Return REST response
    json.NewEncoder(w).Encode(response.Result)
}
```

### 3. WebSocket Bridge
```javascript
// WebSocket to MCP bridge
const WebSocket = require('ws');
const { spawn } = require('child_process');

const wss = new WebSocket.Server({ port: 8080 });
const mcp = spawn('./mcp-server');

wss.on('connection', (ws) => {
    ws.on('message', (message) => {
        // Forward to MCP
        mcp.stdin.write(message + '\n');
    });
    
    mcp.stdout.on('data', (data) => {
        // Forward to WebSocket
        ws.send(data.toString());
    });
});
```

## Best Practices

### 1. Error Handling
```go
func (s *MCPServer) executeToolSafely(name string, args map[string]interface{}) (interface{}, error) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Tool %s panicked: %v", name, r)
        }
    }()
    
    // Validate inputs
    if err := s.validateToolArgs(name, args); err != nil {
        return nil, fmt.Errorf("invalid arguments: %w", err)
    }
    
    // Execute with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    return s.executeTool(ctx, name, args)
}
```

### 2. Logging
```go
func (s *MCPServer) logRequest(req Request) {
    log.Printf("[REQUEST] ID: %v, Method: %s", req.ID, req.Method)
}

func (s *MCPServer) logResponse(resp Response) {
    if resp.Error != nil {
        log.Printf("[ERROR] ID: %v, Error: %s", resp.ID, resp.Error.Message)
    } else {
        log.Printf("[RESPONSE] ID: %v, Success", resp.ID)
    }
}
```

### 3. Security Considerations
- **Input validation**: Always validate tool arguments
- **Path traversal**: Sanitize file paths
- **Rate limiting**: Implement request throttling
- **Authentication**: Add client verification if needed
- **Sandboxing**: Run tools in restricted environments

### 4. Performance Tips
- **Concurrent requests**: Handle multiple requests in parallel
- **Caching**: Cache frequently accessed data
- **Streaming**: Use streaming for large responses
- **Resource limits**: Set memory and CPU limits

### 5. Tool Design Guidelines
- **Single purpose**: Each tool should do one thing well
- **Clear naming**: Use descriptive, action-based names
- **Rich descriptions**: Help AI understand tool usage
- **Error messages**: Provide helpful error context
- **Idempotency**: Make tools safe to retry

## Example: Complete MCP Server

Here's a minimal but complete MCP server:

```go
package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "log"
    "os"
)

type Server struct {
    name    string
    version string
}

func NewServer() *Server {
    return &Server{
        name:    "example-mcp-server",
        version: "1.0.0",
    }
}

func (s *Server) Run() {
    scanner := bufio.NewScanner(os.Stdin)
    encoder := json.NewEncoder(os.Stdout)
    
    log.Println("MCP Server started")
    
    for scanner.Scan() {
        var req map[string]interface{}
        if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
            log.Printf("Failed to parse request: %v", err)
            continue
        }
        
        response := s.handleRequest(req)
        if err := encoder.Encode(response); err != nil {
            log.Printf("Failed to encode response: %v", err)
        }
    }
}

func (s *Server) handleRequest(req map[string]interface{}) map[string]interface{} {
    method, _ := req["method"].(string)
    id := req["id"]
    
    switch method {
    case "initialize":
        return s.initialize(id)
    case "tools/list":
        return s.listTools(id)
    case "tools/call":
        return s.callTool(req, id)
    default:
        return s.errorResponse(id, -32601, "Method not found")
    }
}

func (s *Server) initialize(id interface{}) map[string]interface{} {
    return map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      id,
        "result": map[string]interface{}{
            "protocolVersion": "2024-11-05",
            "capabilities": map[string]interface{}{
                "tools": map[string]interface{}{},
            },
            "serverInfo": map[string]interface{}{
                "name":    s.name,
                "version": s.version,
            },
        },
    }
}

func (s *Server) listTools(id interface{}) map[string]interface{} {
    return map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      id,
        "result": map[string]interface{}{
            "tools": []map[string]interface{}{
                {
                    "name":        "echo",
                    "description": "Echo back the input message",
                    "inputSchema": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "message": map[string]interface{}{
                                "type":        "string",
                                "description": "Message to echo",
                            },
                        },
                        "required": []string{"message"},
                    },
                },
            },
        },
    }
}

func (s *Server) callTool(req map[string]interface{}, id interface{}) map[string]interface{} {
    params, _ := req["params"].(map[string]interface{})
    toolName, _ := params["name"].(string)
    arguments, _ := params["arguments"].(string)
    
    var args map[string]interface{}
    json.Unmarshal([]byte(arguments), &args)
    
    switch toolName {
    case "echo":
        message, _ := args["message"].(string)
        return map[string]interface{}{
            "jsonrpc": "2.0",
            "id":      id,
            "result": map[string]interface{}{
                "content": []map[string]interface{}{
                    {
                        "type": "text",
                        "text": fmt.Sprintf("Echo: %s", message),
                    },
                },
            },
        }
    default:
        return s.errorResponse(id, -32601, "Unknown tool")
    }
}

func (s *Server) errorResponse(id interface{}, code int, message string) map[string]interface{} {
    return map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      id,
        "error": map[string]interface{}{
            "code":    code,
            "message": message,
        },
    }
}

func main() {
    server := NewServer()
    server.Run()
}
```

## Debugging Tips

1. **Enable verbose logging**: Log all requests and responses
2. **Use error codes**: Follow JSON-RPC error code conventions
3. **Test with curl**: Manually send requests to debug
4. **Check message format**: Ensure proper JSON formatting
5. **Monitor stderr**: MCP servers can log to stderr

## Resources

- [JSON-RPC 2.0 Specification](https://www.jsonrpc.org/specification)
- [MCP Protocol Documentation](https://modelcontextprotocol.io)
- Example implementations in various languages
- Community tools and libraries

## Conclusion

Building an MCP server allows you to extend AI capabilities with custom tools and integrations. Focus on:
- Clean tool design
- Robust error handling
- Clear documentation
- Comprehensive testing

Start simple with basic tools and gradually add complexity as needed.