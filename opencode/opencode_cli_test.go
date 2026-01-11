package opencode_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/randalmurphal/llmkit/opencode"
	"github.com/randalmurphal/llmkit/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenCodeCLI_Options(t *testing.T) {
	// Test WithOpenCodePath
	client := opencode.NewOpenCodeCLI(opencode.WithOpenCodePath("/custom/path/opencode"))
	assert.NotNil(t, client)

	// Test WithWorkdir
	client = opencode.NewOpenCodeCLI(opencode.WithWorkdir("/some/workdir"))
	assert.NotNil(t, client)

	// Test WithAllowedTools
	client = opencode.NewOpenCodeCLI(opencode.WithAllowedTools([]string{"write", "edit"}))
	assert.NotNil(t, client)

	// Test all options combined
	client = opencode.NewOpenCodeCLI(
		opencode.WithOpenCodePath("/custom/opencode"),
		opencode.WithWorkdir("/project"),
		opencode.WithAllowedTools([]string{"bash"}),
		opencode.WithTimeout(10*time.Minute),
	)
	assert.NotNil(t, client)
}

func TestOpenCodeCLI_OutputOptions(t *testing.T) {
	t.Run("output format options", func(t *testing.T) {
		client := opencode.NewOpenCodeCLI(
			opencode.WithOutputFormat(opencode.OutputFormatJSON),
		)
		assert.NotNil(t, client)

		client = opencode.NewOpenCodeCLI(
			opencode.WithOutputFormat(opencode.OutputFormatText),
		)
		assert.NotNil(t, client)
	})

	t.Run("quiet mode", func(t *testing.T) {
		client := opencode.NewOpenCodeCLI(opencode.WithQuiet(true))
		assert.NotNil(t, client)

		client = opencode.NewOpenCodeCLI(opencode.WithQuiet(false))
		assert.NotNil(t, client)
	})

	t.Run("debug mode", func(t *testing.T) {
		client := opencode.NewOpenCodeCLI(opencode.WithDebug(true))
		assert.NotNil(t, client)
	})
}

func TestOpenCodeCLI_AgentOptions(t *testing.T) {
	t.Run("build agent", func(t *testing.T) {
		client := opencode.NewOpenCodeCLI(opencode.WithAgent(opencode.AgentBuild))
		assert.NotNil(t, client)
	})

	t.Run("plan agent", func(t *testing.T) {
		client := opencode.NewOpenCodeCLI(opencode.WithAgent(opencode.AgentPlan))
		assert.NotNil(t, client)
	})
}

func TestOpenCodeCLI_ToolControlOptions(t *testing.T) {
	client := opencode.NewOpenCodeCLI(
		opencode.WithAllowedTools([]string{"write", "edit"}),
		opencode.WithDisallowedTools([]string{"bash"}),
	)
	assert.NotNil(t, client)
}

func TestOpenCodeCLI_PromptOptions(t *testing.T) {
	client := opencode.NewOpenCodeCLI(
		opencode.WithSystemPrompt("You are a helpful assistant"),
	)
	assert.NotNil(t, client)
}

func TestOpenCodeCLI_LimitOptions(t *testing.T) {
	client := opencode.NewOpenCodeCLI(opencode.WithMaxTurns(10))
	assert.NotNil(t, client)
}

func TestOpenCodeCLI_EnvOptions(t *testing.T) {
	client := opencode.NewOpenCodeCLI(
		opencode.WithEnv(map[string]string{"KEY": "value"}),
	)
	assert.NotNil(t, client)

	client = opencode.NewOpenCodeCLI(
		opencode.WithEnvVar("SINGLE_KEY", "single_value"),
	)
	assert.NotNil(t, client)
}

func TestOpenCodeCLI_ProductionConfig(t *testing.T) {
	client := opencode.NewOpenCodeCLI(
		opencode.WithOpenCodePath("/usr/local/bin/opencode"),
		opencode.WithWorkdir("/home/user/project"),
		opencode.WithTimeout(10*time.Minute),
		opencode.WithOutputFormat(opencode.OutputFormatJSON),
		opencode.WithQuiet(true),
		opencode.WithAgent(opencode.AgentBuild),
		opencode.WithMaxTurns(20),
		opencode.WithDisallowedTools([]string{"bash"}),
		opencode.WithSystemPrompt("Be extra careful with code changes"),
	)
	assert.NotNil(t, client)
}

func TestOpenCodeCLI_OutputFormatConstants(t *testing.T) {
	assert.Equal(t, opencode.OutputFormat("text"), opencode.OutputFormatText)
	assert.Equal(t, opencode.OutputFormat("json"), opencode.OutputFormatJSON)
}

func TestOpenCodeCLI_AgentConstants(t *testing.T) {
	assert.Equal(t, opencode.Agent("build"), opencode.AgentBuild)
	assert.Equal(t, opencode.Agent("plan"), opencode.AgentPlan)
}

func TestCompletionResponse_Fields(t *testing.T) {
	resp := &opencode.CompletionResponse{
		Content:      "Hello",
		FinishReason: "stop",
		Model:        "gpt-4",
		Usage: opencode.TokenUsage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
	}

	assert.Equal(t, "Hello", resp.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, "gpt-4", resp.Model)
	assert.Equal(t, 150, resp.Usage.TotalTokens)
}

func TestTokenUsage_Add(t *testing.T) {
	usage := opencode.TokenUsage{
		InputTokens:  10,
		OutputTokens: 5,
		TotalTokens:  15,
	}

	other := opencode.TokenUsage{
		InputTokens:  20,
		OutputTokens: 10,
		TotalTokens:  30,
	}

	usage.Add(other)

	assert.Equal(t, 30, usage.InputTokens)
	assert.Equal(t, 15, usage.OutputTokens)
	assert.Equal(t, 45, usage.TotalTokens)
}

func TestOpenCodeCLI_IntegrationSkip(t *testing.T) {
	// Skip if opencode binary not available
	if _, err := exec.LookPath("opencode"); err != nil {
		t.Skip("opencode binary not available, skipping integration test")
	}

	client := opencode.NewOpenCodeCLI()
	assert.NotNil(t, client)
}

func TestOpenCodeCLI_Error(t *testing.T) {
	err := opencode.NewError("complete", assert.AnError, true)
	assert.Contains(t, err.Error(), "opencode complete")
	assert.True(t, err.Retryable)
	assert.Equal(t, assert.AnError, err.Unwrap())
}

func TestLLMErrors(t *testing.T) {
	assert.NotNil(t, opencode.ErrUnavailable)
	assert.NotNil(t, opencode.ErrContextTooLong)
	assert.NotNil(t, opencode.ErrRateLimited)
	assert.NotNil(t, opencode.ErrInvalidRequest)
	assert.NotNil(t, opencode.ErrTimeout)
}

func TestOpenCodeCLI_WithTimeout(t *testing.T) {
	client := opencode.NewOpenCodeCLI(opencode.WithTimeout(10 * time.Second))
	assert.NotNil(t, client)
}

func TestOpenCodeCLI_Complete_NonExistentBinary(t *testing.T) {
	client := opencode.NewOpenCodeCLI(opencode.WithOpenCodePath("/nonexistent/path/to/opencode"))

	_, err := client.Complete(context.Background(), opencode.CompletionRequest{
		Messages: []opencode.Message{{Role: opencode.RoleUser, Content: "test"}},
	})

	assert.Error(t, err)
}

func TestOpenCodeCLI_Stream_NonExistentBinary(t *testing.T) {
	client := opencode.NewOpenCodeCLI(opencode.WithOpenCodePath("/nonexistent/path/to/opencode"))

	_, err := client.Stream(context.Background(), opencode.CompletionRequest{
		Messages: []opencode.Message{{Role: opencode.RoleUser, Content: "test"}},
	})

	assert.Error(t, err)
}

func TestOpenCodeCLI_Provider(t *testing.T) {
	client := opencode.NewOpenCodeCLI()
	assert.Equal(t, "opencode", client.Provider())
}

func TestOpenCodeCLI_Capabilities(t *testing.T) {
	client := opencode.NewOpenCodeCLI()
	caps := client.Capabilities()

	assert.True(t, caps.Streaming)
	assert.True(t, caps.Tools)
	assert.True(t, caps.MCP)
	assert.False(t, caps.Sessions) // OpenCode doesn't support sessions
	assert.False(t, caps.Images)   // OpenCode doesn't support images

	// Check native tools
	assert.True(t, caps.HasTool("write"))
	assert.True(t, caps.HasTool("edit"))
	assert.True(t, caps.HasTool("bash"))
	assert.True(t, caps.HasTool("WebFetch"))
	assert.True(t, caps.HasTool("Task"))
	assert.False(t, caps.HasTool("Read")) // Not a native tool
}

func TestOpenCodeCLI_Close(t *testing.T) {
	client := opencode.NewOpenCodeCLI()
	err := client.Close()
	assert.NoError(t, err)
}

// Config tests

func TestConfig_DefaultConfig(t *testing.T) {
	cfg := opencode.DefaultConfig()

	assert.Equal(t, "opencode", cfg.OpenCodePath)
	assert.Equal(t, 5*time.Minute, cfg.Timeout)
	assert.Equal(t, opencode.AgentBuild, cfg.Agent)
	assert.Equal(t, opencode.OutputFormatJSON, cfg.OutputFormat)
	assert.True(t, cfg.Quiet)
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := opencode.DefaultConfig()
		err := cfg.Validate()
		assert.NoError(t, err)
	})

	t.Run("negative timeout", func(t *testing.T) {
		cfg := opencode.DefaultConfig()
		cfg.Timeout = -1
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("negative max_turns", func(t *testing.T) {
		cfg := opencode.DefaultConfig()
		cfg.MaxTurns = -1
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max_turns")
	})

	t.Run("invalid agent", func(t *testing.T) {
		cfg := opencode.DefaultConfig()
		cfg.Agent = "invalid"
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "agent")
	})

	t.Run("invalid output_format", func(t *testing.T) {
		cfg := opencode.DefaultConfig()
		cfg.OutputFormat = "invalid"
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "output_format")
	})
}

func TestConfig_ToOptions(t *testing.T) {
	cfg := opencode.Config{
		OpenCodePath: "/custom/opencode",
		WorkDir:      "/project",
		Timeout:      10 * time.Minute,
		Agent:        opencode.AgentPlan,
		OutputFormat: opencode.OutputFormatJSON,
		Quiet:        true,
		Debug:        true,
		SystemPrompt: "Be helpful",
		MaxTurns:     20,
		AllowedTools: []string{"edit"},
		Env:          map[string]string{"KEY": "value"},
	}

	opts := cfg.ToOptions()
	assert.NotEmpty(t, opts)

	// Verify options work by creating a client
	client := opencode.NewOpenCodeCLI(opts...)
	assert.NotNil(t, client)
}

func TestConfig_NewFromConfig(t *testing.T) {
	cfg := opencode.Config{
		OpenCodePath: "/custom/opencode",
		Timeout:      10 * time.Minute,
	}

	client := opencode.NewFromConfig(cfg)
	assert.NotNil(t, client)
}

func TestConfig_NewFromConfigWithOptions(t *testing.T) {
	cfg := opencode.Config{
		OpenCodePath: "/custom/opencode",
	}

	// Additional options should override config
	client := opencode.NewFromConfig(cfg, opencode.WithTimeout(15*time.Minute))
	assert.NotNil(t, client)
}

func TestConfig_FromEnv(t *testing.T) {
	cfg := opencode.FromEnv()
	assert.NotNil(t, cfg.OpenCodePath)
}

// Provider registration tests

func TestProviderRegistration(t *testing.T) {
	// The opencode provider should be registered via init()
	assert.True(t, provider.IsRegistered("opencode"))
}

func TestProvider_New(t *testing.T) {
	client, err := provider.New("opencode", provider.Config{
		Provider: "opencode",
	})
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "opencode", client.Provider())
	_ = client.Close()
}

func TestProvider_Capabilities(t *testing.T) {
	client, err := provider.New("opencode", provider.Config{
		Provider: "opencode",
	})
	require.NoError(t, err)
	defer func() { _ = client.Close() }()

	caps := client.Capabilities()
	assert.True(t, caps.Streaming)
	assert.True(t, caps.Tools)
	assert.True(t, caps.MCP)
	assert.False(t, caps.Sessions)
	assert.False(t, caps.Images)
}

func TestProvider_WithOptions(t *testing.T) {
	client, err := provider.New("opencode", provider.Config{
		Provider:     "opencode",
		SystemPrompt: "Be helpful",
		MaxTurns:     10,
		Timeout:      10 * time.Minute,
		WorkDir:      "/project",
		Options: map[string]any{
			"quiet": true,
			"agent": "plan",
			"debug": false,
		},
	})
	require.NoError(t, err)
	assert.NotNil(t, client)
	_ = client.Close()
}

func TestCapabilities_HasTool(t *testing.T) {
	caps := opencode.Capabilities{
		NativeTools: []string{"write", "edit", "bash"},
	}

	assert.True(t, caps.HasTool("write"))
	assert.True(t, caps.HasTool("edit"))
	assert.True(t, caps.HasTool("bash"))
	assert.False(t, caps.HasTool("nonexistent"))
}
