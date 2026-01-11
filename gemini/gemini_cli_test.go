package gemini_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/randalmurphal/llmkit/gemini"
	"github.com/stretchr/testify/assert"
)

func TestGeminiCLI_Options(t *testing.T) {
	// Test WithGeminiPath
	client := gemini.NewGeminiCLI(gemini.WithGeminiPath("/custom/path/gemini"))
	assert.NotNil(t, client)

	// Test WithWorkdir
	client = gemini.NewGeminiCLI(gemini.WithWorkdir("/some/workdir"))
	assert.NotNil(t, client)

	// Test WithAllowedTools
	client = gemini.NewGeminiCLI(gemini.WithAllowedTools([]string{"read_file", "write_file"}))
	assert.NotNil(t, client)

	// Test all options combined
	client = gemini.NewGeminiCLI(
		gemini.WithGeminiPath("/custom/gemini"),
		gemini.WithModel("gemini-2.5-pro"),
		gemini.WithWorkdir("/project"),
		gemini.WithAllowedTools([]string{"run_shell_command"}),
	)
	assert.NotNil(t, client)
}

func TestGeminiCLI_NewOptions(t *testing.T) {
	// Test output control options
	t.Run("output format options", func(t *testing.T) {
		client := gemini.NewGeminiCLI(
			gemini.WithOutputFormat(gemini.OutputFormatJSON),
		)
		assert.NotNil(t, client)
	})

	// Test tool control options
	t.Run("tool control options", func(t *testing.T) {
		// Note: Gemini CLI only supports --allowed-tools, not --disallowed-tools
		client := gemini.NewGeminiCLI(
			gemini.WithAllowedTools([]string{"read_file", "write_file"}),
		)
		assert.NotNil(t, client)
	})

	// Test permission options
	t.Run("permission options", func(t *testing.T) {
		client := gemini.NewGeminiCLI(gemini.WithYolo())
		assert.NotNil(t, client)
	})

	// Test context options
	t.Run("context options", func(t *testing.T) {
		client := gemini.NewGeminiCLI(
			gemini.WithIncludeDirs([]string{"/tmp", "/home/user/project"}),
		)
		assert.NotNil(t, client)

		client = gemini.NewGeminiCLI(gemini.WithSystemPrompt("You are a helpful assistant"))
		assert.NotNil(t, client)
	})

	// Test sandbox options
	t.Run("sandbox options", func(t *testing.T) {
		client := gemini.NewGeminiCLI(gemini.WithSandbox("docker"))
		assert.NotNil(t, client)

		client = gemini.NewGeminiCLI(gemini.WithSandbox("host"))
		assert.NotNil(t, client)
	})

	// Test environment options
	t.Run("environment options", func(t *testing.T) {
		client := gemini.NewGeminiCLI(gemini.WithEnv(map[string]string{
			"GOOGLE_API_KEY": "test-key",
		}))
		assert.NotNil(t, client)

		client = gemini.NewGeminiCLI(gemini.WithEnvVar("GOOGLE_API_KEY", "test-key"))
		assert.NotNil(t, client)
	})

	// Test MCP options
	t.Run("MCP options", func(t *testing.T) {
		client := gemini.NewGeminiCLI(gemini.WithMCPConfig("/path/to/mcp.json"))
		assert.NotNil(t, client)

		client = gemini.NewGeminiCLI(gemini.WithMCPServers(map[string]gemini.MCPServerConfig{
			"test-server": {
				Type:    "stdio",
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/test-server"},
			},
		}))
		assert.NotNil(t, client)
	})

	// Test production configuration (all options combined)
	t.Run("production configuration", func(t *testing.T) {
		client := gemini.NewGeminiCLI(
			gemini.WithGeminiPath("/usr/local/bin/gemini"),
			gemini.WithModel("gemini-2.5-pro"),
			gemini.WithWorkdir("/home/user/project"),
			gemini.WithTimeout(10*time.Minute),
			gemini.WithOutputFormat(gemini.OutputFormatJSON),
			gemini.WithYolo(),
			gemini.WithSandbox("docker"),
			gemini.WithAllowedTools([]string{"read_file", "write_file"}),
			gemini.WithSystemPrompt("Be extra careful with code changes"),
		)
		assert.NotNil(t, client)
	})
}

func TestGeminiCLI_OutputFormatConstants(t *testing.T) {
	// Verify output format constants are accessible
	assert.Equal(t, gemini.OutputFormat("text"), gemini.OutputFormatText)
	assert.Equal(t, gemini.OutputFormat("json"), gemini.OutputFormatJSON)
	assert.Equal(t, gemini.OutputFormat("stream-json"), gemini.OutputFormatStreamJSON)
}

func TestCompletionResponse_Fields(t *testing.T) {
	// Test that fields are accessible on CompletionResponse
	resp := &gemini.CompletionResponse{
		Content:      "Hello",
		NumTurns:     2,
		FinishReason: "stop",
		Model:        "gemini-2.5-pro",
		Usage: gemini.TokenUsage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
	}

	assert.Equal(t, "Hello", resp.Content)
	assert.Equal(t, 2, resp.NumTurns)
	assert.Equal(t, 150, resp.Usage.TotalTokens)
}

func TestTokenUsage_Add(t *testing.T) {
	usage := gemini.TokenUsage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}

	other := gemini.TokenUsage{
		InputTokens:  200,
		OutputTokens: 100,
		TotalTokens:  300,
	}

	usage.Add(other)

	assert.Equal(t, 300, usage.InputTokens)
	assert.Equal(t, 150, usage.OutputTokens)
	assert.Equal(t, 450, usage.TotalTokens)
}

func TestGeminiCLI_IntegrationSkip(t *testing.T) {
	// Skip if gemini binary not available
	if _, err := exec.LookPath("gemini"); err != nil {
		t.Skip("gemini binary not available, skipping integration test")
	}

	// This would be an actual integration test if gemini is available
	// For now, just verify the client can be created
	client := gemini.NewGeminiCLI()
	assert.NotNil(t, client)
}

func TestGeminiCLI_Error(t *testing.T) {
	err := gemini.NewError("complete", assert.AnError, true)
	assert.Contains(t, err.Error(), "gemini complete")
	assert.True(t, err.Retryable)
	assert.Equal(t, assert.AnError, err.Unwrap())
}

func TestLLMErrors(t *testing.T) {
	// Verify sentinel errors are defined
	assert.NotNil(t, gemini.ErrUnavailable)
	assert.NotNil(t, gemini.ErrContextTooLong)
	assert.NotNil(t, gemini.ErrRateLimited)
	assert.NotNil(t, gemini.ErrInvalidRequest)
	assert.NotNil(t, gemini.ErrTimeout)
	assert.NotNil(t, gemini.ErrQuotaExceeded)
}

func TestGeminiCLI_WithTimeout(t *testing.T) {
	client := gemini.NewGeminiCLI(gemini.WithTimeout(10 * time.Second))
	assert.NotNil(t, client)
}

func TestGeminiCLI_Complete_NonExistentBinary(t *testing.T) {
	client := gemini.NewGeminiCLI(gemini.WithGeminiPath("/nonexistent/path/to/gemini"))

	_, err := client.Complete(context.Background(), gemini.CompletionRequest{
		Messages: []gemini.Message{{Role: gemini.RoleUser, Content: "test"}},
	})

	assert.Error(t, err)
}

func TestGeminiCLI_Stream_NonExistentBinary(t *testing.T) {
	client := gemini.NewGeminiCLI(gemini.WithGeminiPath("/nonexistent/path/to/gemini"))

	_, err := client.Stream(context.Background(), gemini.CompletionRequest{
		Messages: []gemini.Message{{Role: gemini.RoleUser, Content: "test"}},
	})

	assert.Error(t, err)
}

func TestGeminiCLI_Provider(t *testing.T) {
	client := gemini.NewGeminiCLI()
	assert.Equal(t, "gemini", client.Provider())
}

func TestGeminiCLI_Capabilities(t *testing.T) {
	client := gemini.NewGeminiCLI()
	caps := client.Capabilities()

	assert.True(t, caps.Streaming)
	assert.True(t, caps.Tools)
	assert.True(t, caps.MCP)
	assert.False(t, caps.Sessions) // Gemini doesn't support sessions like Claude
	assert.True(t, caps.Images)
	assert.Equal(t, "GEMINI.md", caps.ContextFile)

	// Check native tools
	assert.True(t, caps.HasTool("read_file"))
	assert.True(t, caps.HasTool("write_file"))
	assert.True(t, caps.HasTool("run_shell_command"))
	assert.True(t, caps.HasTool("web_fetch"))
	assert.True(t, caps.HasTool("google_web_search"))
	assert.True(t, caps.HasTool("save_memory"))
	assert.True(t, caps.HasTool("write_todos"))
	assert.False(t, caps.HasTool("nonexistent_tool"))
}

func TestGeminiCLI_Close(t *testing.T) {
	client := gemini.NewGeminiCLI()
	err := client.Close()
	assert.NoError(t, err)
}

func TestConfig_DefaultConfig(t *testing.T) {
	cfg := gemini.DefaultConfig()
	assert.Equal(t, "gemini-2.5-pro", cfg.Model)
	assert.Equal(t, 5*time.Minute, cfg.Timeout)
	assert.Equal(t, gemini.OutputFormatJSON, cfg.OutputFormat)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     gemini.Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     gemini.DefaultConfig(),
			wantErr: false,
		},
		{
			name: "missing model",
			cfg: gemini.Config{
				Model: "",
			},
			wantErr: true,
		},
		{
			name: "negative max turns",
			cfg: gemini.Config{
				Model:    "gemini-2.5-pro",
				MaxTurns: -1,
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			cfg: gemini.Config{
				Model:   "gemini-2.5-pro",
				Timeout: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "invalid sandbox",
			cfg: gemini.Config{
				Model:   "gemini-2.5-pro",
				Sandbox: "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid sandbox - docker",
			cfg: gemini.Config{
				Model:   "gemini-2.5-pro",
				Sandbox: "docker",
			},
			wantErr: false,
		},
		{
			name: "valid sandbox - host",
			cfg: gemini.Config{
				Model:   "gemini-2.5-pro",
				Sandbox: "host",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_ToOptions(t *testing.T) {
	cfg := gemini.Config{
		Model:        "gemini-2.5-pro",
		SystemPrompt: "You are helpful",
		MaxTurns:     10,
		Timeout:      5 * time.Minute,
		WorkDir:      "/project",
		AllowedTools: []string{"read_file", "write_file"},
		Yolo:         true,
		OutputFormat: gemini.OutputFormatJSON,
		IncludeDirs:  []string{"/extra"},
		Sandbox:      "docker",
		GeminiPath:   "/usr/bin/gemini",
	}

	opts := cfg.ToOptions()
	assert.NotEmpty(t, opts)

	// Verify options can be applied
	client := gemini.NewGeminiCLI(opts...)
	assert.NotNil(t, client)
}

func TestMessage_MultimodalContent(t *testing.T) {
	// Test text-only message
	textMsg := gemini.Message{
		Role:    gemini.RoleUser,
		Content: "Hello",
	}
	assert.Equal(t, "Hello", textMsg.Content)

	// Test multimodal message with image
	imageMsg := gemini.Message{
		Role: gemini.RoleUser,
		ContentParts: []gemini.ContentPart{
			{Type: "text", Text: "What's in this image?"},
			{Type: "image", ImageURL: "https://example.com/image.png"},
		},
	}
	assert.Len(t, imageMsg.ContentParts, 2)
	assert.Equal(t, "text", imageMsg.ContentParts[0].Type)
	assert.Equal(t, "image", imageMsg.ContentParts[1].Type)

	// Test multimodal message with base64 image
	base64Msg := gemini.Message{
		Role: gemini.RoleUser,
		ContentParts: []gemini.ContentPart{
			{Type: "text", Text: "Analyze this"},
			{Type: "image", ImageBase64: "iVBORw0KGgo...", MediaType: "image/png"},
		},
	}
	assert.Equal(t, "image/png", base64Msg.ContentParts[1].MediaType)
}

func TestCapabilities_HasTool(t *testing.T) {
	caps := gemini.Capabilities{
		NativeTools: []string{"read_file", "write_file", "run_shell_command"},
	}

	assert.True(t, caps.HasTool("read_file"))
	assert.True(t, caps.HasTool("write_file"))
	assert.True(t, caps.HasTool("run_shell_command"))
	assert.False(t, caps.HasTool("nonexistent"))
}

func TestRole_Constants(t *testing.T) {
	assert.Equal(t, gemini.Role("user"), gemini.RoleUser)
	assert.Equal(t, gemini.Role("assistant"), gemini.RoleAssistant)
	assert.Equal(t, gemini.Role("tool"), gemini.RoleTool)
	assert.Equal(t, gemini.Role("system"), gemini.RoleSystem)
}
