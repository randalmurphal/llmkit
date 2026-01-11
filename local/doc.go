// Package local provides a client for local LLM models via a Python sidecar process.
//
// The local provider communicates with a Python sidecar via JSON-RPC 2.0 over stdio.
// This enables llmkit to work with local model backends like Ollama, llama.cpp, vLLM,
// and HuggingFace transformers without requiring direct Go bindings.
//
// # Architecture
//
// The local client manages a long-running Python sidecar process:
//
//	Go Client <--JSON-RPC/stdio--> Python Sidecar <--Backend API--> Local Model
//
// The sidecar process is started lazily on the first request and is kept running
// for subsequent requests. The sidecar handles all communication with the local
// model backend and can optionally connect to MCP servers.
//
// # Supported Backends
//
//   - ollama: Ollama API server (default host: localhost:11434)
//   - llama.cpp: llama.cpp server (default host: localhost:8000)
//   - vllm: vLLM server (default host: localhost:8000)
//   - transformers: HuggingFace transformers (runs in-process in sidecar)
//
// # JSON-RPC Protocol
//
// Communication uses JSON-RPC 2.0 over stdio with newline-delimited messages:
//
// Request (client -> sidecar):
//
//	{"jsonrpc": "2.0", "method": "complete", "params": {...}, "id": 1}
//
// Response (sidecar -> client):
//
//	{"jsonrpc": "2.0", "result": {"content": "...", "usage": {...}}, "id": 1}
//
// Streaming uses notifications (no ID, no response expected):
//
//	{"jsonrpc": "2.0", "method": "stream.chunk", "params": {"content": "...", "done": false}}
//	{"jsonrpc": "2.0", "method": "stream.done", "params": {"usage": {...}}}
//
// # Usage
//
// Using the provider registry:
//
//	import _ "github.com/randalmurphal/llmkit/local"
//
//	client, err := provider.New("local", provider.Config{
//	    Model: "llama3.2:latest",
//	    Options: map[string]any{
//	        "backend":      "ollama",
//	        "sidecar_path": "/path/to/sidecar.py",
//	    },
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
// Direct instantiation:
//
//	client := local.NewClient(
//	    local.WithBackend(local.BackendOllama),
//	    local.WithSidecarPath("/path/to/sidecar.py"),
//	    local.WithModel("llama3.2:latest"),
//	)
//	defer client.Close()
//
//	resp, err := client.Complete(ctx, provider.Request{
//	    Messages: []provider.Message{
//	        {Role: provider.RoleUser, Content: "Hello!"},
//	    },
//	})
//
// # Capabilities
//
// The local provider has the following capabilities:
//
//   - Streaming: true (via JSON-RPC notifications)
//   - Tools: false (local models don't have native tool support)
//   - MCP: true (the sidecar can connect to MCP servers)
//   - Sessions: false (no persistent sessions)
//   - Images: false (multimodal not currently supported)
//   - NativeTools: none
//
// # Sidecar Implementation
//
// The Python sidecar script must implement the following RPC methods:
//
//   - init: Initialize the backend connection
//   - complete: Perform a completion request
//   - shutdown: Clean shutdown
//
// For streaming, the sidecar should send stream.chunk and stream.done notifications.
// See the protocol.go file for detailed message formats.
package local
