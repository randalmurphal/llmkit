package local

import (
	"context"
	"testing"
	"time"

	"github.com/randalmurphal/llmkit/provider"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Backend:     BackendOllama,
				SidecarPath: "/path/to/sidecar.py",
				Model:       "llama3.2:latest",
			},
			wantErr: false,
		},
		{
			name: "missing backend",
			cfg: Config{
				SidecarPath: "/path/to/sidecar.py",
				Model:       "llama3.2:latest",
			},
			wantErr: true,
		},
		{
			name: "missing sidecar path",
			cfg: Config{
				Backend: BackendOllama,
				Model:   "llama3.2:latest",
			},
			wantErr: true,
		},
		{
			name: "missing model",
			cfg: Config{
				Backend:     BackendOllama,
				SidecarPath: "/path/to/sidecar.py",
			},
			wantErr: true,
		},
		{
			name: "invalid backend",
			cfg: Config{
				Backend:     Backend("unknown"),
				SidecarPath: "/path/to/sidecar.py",
				Model:       "llama3.2:latest",
			},
			wantErr: true,
		},
		{
			name: "negative startup timeout",
			cfg: Config{
				Backend:        BackendOllama,
				SidecarPath:    "/path/to/sidecar.py",
				Model:          "llama3.2:latest",
				StartupTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "all backends valid",
			cfg: Config{
				Backend:     BackendVLLM,
				SidecarPath: "/path/to/sidecar.py",
				Model:       "meta-llama/Llama-3.2-8B",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_WithDefaults(t *testing.T) {
	cfg := Config{
		Backend:     BackendOllama,
		SidecarPath: "/path/to/sidecar.py",
		Model:       "llama3.2:latest",
	}

	result := cfg.WithDefaults()

	if result.PythonPath != "python3" {
		t.Errorf("PythonPath = %q, want %q", result.PythonPath, "python3")
	}
	if result.StartupTimeout != 30*time.Second {
		t.Errorf("StartupTimeout = %v, want %v", result.StartupTimeout, 30*time.Second)
	}
	if result.RequestTimeout != 5*time.Minute {
		t.Errorf("RequestTimeout = %v, want %v", result.RequestTimeout, 5*time.Minute)
	}
	if result.Host != "localhost:11434" {
		t.Errorf("Host = %q, want %q", result.Host, "localhost:11434")
	}
}

func TestConfig_WithDefaults_VLLM(t *testing.T) {
	cfg := Config{
		Backend:     BackendVLLM,
		SidecarPath: "/path/to/sidecar.py",
		Model:       "model",
	}

	result := cfg.WithDefaults()

	if result.Host != "localhost:8000" {
		t.Errorf("Host = %q, want %q for vLLM", result.Host, "localhost:8000")
	}
}

func TestClient_Capabilities(t *testing.T) {
	client := NewClient(
		WithBackend(BackendOllama),
		WithSidecarPath("/path/to/sidecar.py"),
		WithModel("llama3.2:latest"),
	)

	caps := client.Capabilities()

	if !caps.Streaming {
		t.Error("Streaming should be true")
	}
	if caps.Tools {
		t.Error("Tools should be false for local models")
	}
	if !caps.MCP {
		t.Error("MCP should be true (via sidecar)")
	}
	if caps.Sessions {
		t.Error("Sessions should be false for local models")
	}
	if caps.Images {
		t.Error("Images should be false for local models")
	}
	if len(caps.NativeTools) != 0 {
		t.Errorf("NativeTools should be empty, got %v", caps.NativeTools)
	}
}

func TestClient_Provider(t *testing.T) {
	client := NewClient()
	if client.Provider() != "local" {
		t.Errorf("Provider() = %q, want %q", client.Provider(), "local")
	}
}

func TestNewClient_Options(t *testing.T) {
	env := map[string]string{"KEY": "value"}
	mcpServers := map[string]MCPServerConfig{
		"test": {Type: "stdio", Command: "cmd"},
	}

	client := NewClient(
		WithBackend(BackendLlamaCpp),
		WithSidecarPath("/custom/sidecar.py"),
		WithModel("llama-3.2"),
		WithHost("localhost:9000"),
		WithPythonPath("/usr/bin/python"),
		WithStartupTimeout(60*time.Second),
		WithRequestTimeout(10*time.Minute),
		WithWorkDir("/work"),
		WithEnv(env),
		WithMCPServers(mcpServers),
	)

	cfg := client.cfg

	if cfg.Backend != BackendLlamaCpp {
		t.Errorf("Backend = %q, want %q", cfg.Backend, BackendLlamaCpp)
	}
	if cfg.SidecarPath != "/custom/sidecar.py" {
		t.Errorf("SidecarPath = %q, want %q", cfg.SidecarPath, "/custom/sidecar.py")
	}
	if cfg.Model != "llama-3.2" {
		t.Errorf("Model = %q, want %q", cfg.Model, "llama-3.2")
	}
	if cfg.Host != "localhost:9000" {
		t.Errorf("Host = %q, want %q", cfg.Host, "localhost:9000")
	}
	if cfg.PythonPath != "/usr/bin/python" {
		t.Errorf("PythonPath = %q, want %q", cfg.PythonPath, "/usr/bin/python")
	}
	if cfg.StartupTimeout != 60*time.Second {
		t.Errorf("StartupTimeout = %v, want %v", cfg.StartupTimeout, 60*time.Second)
	}
	if cfg.RequestTimeout != 10*time.Minute {
		t.Errorf("RequestTimeout = %v, want %v", cfg.RequestTimeout, 10*time.Minute)
	}
	if cfg.WorkDir != "/work" {
		t.Errorf("WorkDir = %q, want %q", cfg.WorkDir, "/work")
	}
	if cfg.Env["KEY"] != "value" {
		t.Errorf("Env[KEY] = %q, want %q", cfg.Env["KEY"], "value")
	}
	if cfg.MCPServers["test"].Command != "cmd" {
		t.Errorf("MCPServers[test].Command = %q, want %q", cfg.MCPServers["test"].Command, "cmd")
	}
}

func TestNewClientWithConfig(t *testing.T) {
	cfg := Config{
		Backend:     BackendTransformers,
		SidecarPath: "/sidecar.py",
		Model:       "model",
	}

	client := NewClientWithConfig(cfg)

	// Should have defaults applied
	if client.cfg.PythonPath != "python3" {
		t.Errorf("PythonPath = %q, want %q", client.cfg.PythonPath, "python3")
	}
}

func TestClient_Complete_NotStarted(t *testing.T) {
	// Client without valid config - should fail validation
	client := NewClient()

	ctx := context.Background()
	req := provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "test"},
		},
	}

	_, err := client.Complete(ctx, req)
	if err == nil {
		t.Error("Complete() should fail with invalid config")
	}
}

func TestClient_Stream_NotStarted(t *testing.T) {
	// Client without valid config - should fail validation
	client := NewClient()

	ctx := context.Background()
	req := provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "test"},
		},
	}

	_, err := client.Stream(ctx, req)
	if err == nil {
		t.Error("Stream() should fail with invalid config")
	}
}

func TestClient_Close_NotStarted(t *testing.T) {
	client := NewClient()

	// Should not error when sidecar was never started
	if err := client.Close(); err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestLocalProviderAdapter_Provider(t *testing.T) {
	adapter := &localProviderAdapter{
		client: NewClient(),
	}

	if adapter.Provider() != "local" {
		t.Errorf("Provider() = %q, want %q", adapter.Provider(), "local")
	}
}

func TestLocalProviderAdapter_Capabilities(t *testing.T) {
	adapter := &localProviderAdapter{
		client: NewClient(),
	}

	caps := adapter.Capabilities()

	// Should match client capabilities
	if !caps.Streaming {
		t.Error("Streaming should be true")
	}
	if caps.Tools {
		t.Error("Tools should be false")
	}
}
