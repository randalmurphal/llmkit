package codex_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/randalmurphal/llmkit/codex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCodexCLI_Options(t *testing.T) {
	// Test WithCodexPath
	client := codex.NewCodexCLI(codex.WithCodexPath("/custom/path/codex"))
	assert.NotNil(t, client)

	// Test WithWorkdir
	client = codex.NewCodexCLI(codex.WithWorkdir("/some/workdir"))
	assert.NotNil(t, client)

	// Test WithModel
	client = codex.NewCodexCLI(codex.WithModel("gpt-5-codex"))
	assert.NotNil(t, client)

	// Test WithTimeout
	client = codex.NewCodexCLI(codex.WithTimeout(10 * time.Minute))
	assert.NotNil(t, client)

	// Test all options combined
	client = codex.NewCodexCLI(
		codex.WithCodexPath("/custom/codex"),
		codex.WithModel("gpt-5-codex"),
		codex.WithWorkdir("/project"),
		codex.WithTimeout(5*time.Minute),
	)
	assert.NotNil(t, client)
}

func TestCodexCLI_SandboxOptions(t *testing.T) {
	t.Run("sandbox modes", func(t *testing.T) {
		client := codex.NewCodexCLI(codex.WithSandboxMode(codex.SandboxReadOnly))
		assert.NotNil(t, client)

		client = codex.NewCodexCLI(codex.WithSandboxMode(codex.SandboxWorkspaceWrite))
		assert.NotNil(t, client)

		client = codex.NewCodexCLI(codex.WithSandboxMode(codex.SandboxDangerFullAccess))
		assert.NotNil(t, client)
	})
}

func TestCodexCLI_ApprovalOptions(t *testing.T) {
	t.Run("approval modes", func(t *testing.T) {
		client := codex.NewCodexCLI(codex.WithApprovalMode(codex.ApprovalUntrusted))
		assert.NotNil(t, client)

		client = codex.NewCodexCLI(codex.WithApprovalMode(codex.ApprovalOnFailure))
		assert.NotNil(t, client)

		client = codex.NewCodexCLI(codex.WithApprovalMode(codex.ApprovalOnRequest))
		assert.NotNil(t, client)

		client = codex.NewCodexCLI(codex.WithApprovalMode(codex.ApprovalNever))
		assert.NotNil(t, client)
	})

	t.Run("full auto", func(t *testing.T) {
		client := codex.NewCodexCLI(codex.WithFullAuto())
		assert.NotNil(t, client)
	})
}

func TestCodexCLI_SessionOptions(t *testing.T) {
	client := codex.NewCodexCLI(codex.WithSessionID("test-session-123"))
	assert.NotNil(t, client)
}

func TestCodexCLI_SearchOption(t *testing.T) {
	client := codex.NewCodexCLI(codex.WithSearch())
	assert.NotNil(t, client)
}

func TestCodexCLI_DirectoryOptions(t *testing.T) {
	t.Run("single directory", func(t *testing.T) {
		client := codex.NewCodexCLI(codex.WithAddDir("/tmp"))
		assert.NotNil(t, client)
	})

	t.Run("multiple directories", func(t *testing.T) {
		client := codex.NewCodexCLI(codex.WithAddDirs([]string{"/tmp", "/home/user"}))
		assert.NotNil(t, client)
	})
}

func TestCodexCLI_ImageOptions(t *testing.T) {
	t.Run("single image", func(t *testing.T) {
		client := codex.NewCodexCLI(codex.WithImage("/path/to/image.png"))
		assert.NotNil(t, client)
	})

	t.Run("multiple images", func(t *testing.T) {
		client := codex.NewCodexCLI(codex.WithImages([]string{"/path/to/img1.png", "/path/to/img2.jpg"}))
		assert.NotNil(t, client)
	})
}

func TestCodexCLI_EnvOptions(t *testing.T) {
	t.Run("env map", func(t *testing.T) {
		client := codex.NewCodexCLI(codex.WithEnv(map[string]string{
			"OPENAI_API_KEY": "test-key",
		}))
		assert.NotNil(t, client)
	})

	t.Run("single env var", func(t *testing.T) {
		client := codex.NewCodexCLI(codex.WithEnvVar("OPENAI_API_KEY", "test-key"))
		assert.NotNil(t, client)
	})
}

func TestCodexCLI_ProductionConfig(t *testing.T) {
	// Test production-like configuration
	client := codex.NewCodexCLI(
		codex.WithCodexPath("/usr/local/bin/codex"),
		codex.WithModel("gpt-5-codex"),
		codex.WithWorkdir("/home/user/project"),
		codex.WithTimeout(10*time.Minute),
		codex.WithSandboxMode(codex.SandboxWorkspaceWrite),
		codex.WithApprovalMode(codex.ApprovalNever),
		codex.WithAddDirs([]string{"/tmp", "/var/log"}),
		codex.WithSearch(),
	)
	assert.NotNil(t, client)
}

func TestCodexCLI_SandboxModeConstants(t *testing.T) {
	assert.Equal(t, codex.SandboxMode("read-only"), codex.SandboxReadOnly)
	assert.Equal(t, codex.SandboxMode("workspace-write"), codex.SandboxWorkspaceWrite)
	assert.Equal(t, codex.SandboxMode("danger-full-access"), codex.SandboxDangerFullAccess)
}

func TestCodexCLI_ApprovalModeConstants(t *testing.T) {
	assert.Equal(t, codex.ApprovalMode("untrusted"), codex.ApprovalUntrusted)
	assert.Equal(t, codex.ApprovalMode("on-failure"), codex.ApprovalOnFailure)
	assert.Equal(t, codex.ApprovalMode("on-request"), codex.ApprovalOnRequest)
	assert.Equal(t, codex.ApprovalMode("never"), codex.ApprovalNever)
}

func TestCompletionResponse_Fields(t *testing.T) {
	resp := &codex.CompletionResponse{
		Content:      "Hello",
		SessionID:    "session-123",
		CostUSD:      0.05,
		NumTurns:     2,
		FinishReason: "stop",
		Model:        "gpt-5-codex",
		Usage: codex.TokenUsage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
	}

	assert.Equal(t, "session-123", resp.SessionID)
	assert.Equal(t, 0.05, resp.CostUSD)
	assert.Equal(t, 2, resp.NumTurns)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 150, resp.Usage.TotalTokens)
}

func TestTokenUsage_Add(t *testing.T) {
	usage := codex.TokenUsage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}

	other := codex.TokenUsage{
		InputTokens:  200,
		OutputTokens: 100,
		TotalTokens:  300,
	}

	usage.Add(other)

	assert.Equal(t, 300, usage.InputTokens)
	assert.Equal(t, 150, usage.OutputTokens)
	assert.Equal(t, 450, usage.TotalTokens)
}

func TestCodexCLI_Error(t *testing.T) {
	err := codex.NewError("complete", assert.AnError, true)
	assert.Contains(t, err.Error(), "codex complete")
	assert.True(t, err.Retryable)
	assert.Equal(t, assert.AnError, err.Unwrap())
}

func TestCodexErrors(t *testing.T) {
	assert.NotNil(t, codex.ErrUnavailable)
	assert.NotNil(t, codex.ErrContextTooLong)
	assert.NotNil(t, codex.ErrRateLimited)
	assert.NotNil(t, codex.ErrInvalidRequest)
	assert.NotNil(t, codex.ErrTimeout)
	assert.NotNil(t, codex.ErrSessionNotFound)
}

func TestCodexCLI_Provider(t *testing.T) {
	client := codex.NewCodexCLI()
	assert.Equal(t, "codex", client.Provider())
}

func TestCodexCLI_Capabilities(t *testing.T) {
	client := codex.NewCodexCLI()
	caps := client.Capabilities()

	assert.True(t, caps.Streaming)
	assert.True(t, caps.Tools)
	assert.True(t, caps.MCP)
	assert.True(t, caps.Sessions)
	assert.True(t, caps.Images)
	assert.Contains(t, caps.NativeTools, "file_read")
	assert.Contains(t, caps.NativeTools, "file_write")
	assert.Contains(t, caps.NativeTools, "shell")
	assert.Contains(t, caps.NativeTools, "web_search")
	assert.Empty(t, caps.ContextFile)
}

func TestCapabilities_HasTool(t *testing.T) {
	caps := codex.Capabilities{
		NativeTools: []string{"file_read", "file_write", "shell"},
	}

	assert.True(t, caps.HasTool("file_read"))
	assert.True(t, caps.HasTool("shell"))
	assert.False(t, caps.HasTool("web_fetch"))
}

func TestCodexCLI_Close(t *testing.T) {
	client := codex.NewCodexCLI()
	err := client.Close()
	assert.NoError(t, err)
}

func TestCodexCLI_Complete_NonExistentBinary(t *testing.T) {
	client := codex.NewCodexCLI(codex.WithCodexPath("/nonexistent/path/to/codex"))

	_, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "test"}},
	})

	assert.Error(t, err)
}

func TestCodexCLI_Stream_NonExistentBinary(t *testing.T) {
	client := codex.NewCodexCLI(codex.WithCodexPath("/nonexistent/path/to/codex"))

	_, err := client.Stream(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "test"}},
	})

	assert.Error(t, err)
}

func TestCodexCLI_IntegrationSkip(t *testing.T) {
	// Skip if codex binary not available
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex binary not available, skipping integration test")
	}

	client := codex.NewCodexCLI()
	assert.NotNil(t, client)
}

func TestConfig_Defaults(t *testing.T) {
	cfg := codex.DefaultConfig()

	assert.Equal(t, 5*time.Minute, cfg.Timeout)
	assert.Equal(t, codex.SandboxWorkspaceWrite, cfg.SandboxMode)
	assert.Empty(t, cfg.Model)
	assert.Empty(t, cfg.WorkDir)
}

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := codex.Config{
			Model:        "gpt-5-codex",
			SandboxMode:  codex.SandboxWorkspaceWrite,
			ApprovalMode: codex.ApprovalNever,
		}
		assert.NoError(t, cfg.Validate())
	})

	t.Run("invalid sandbox mode", func(t *testing.T) {
		cfg := codex.Config{
			SandboxMode: "invalid-mode",
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid sandbox_mode")
	})

	t.Run("invalid approval mode", func(t *testing.T) {
		cfg := codex.Config{
			ApprovalMode: "invalid-mode",
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid approval_mode")
	})

	t.Run("negative timeout", func(t *testing.T) {
		cfg := codex.Config{
			Timeout: -1 * time.Second,
		}
		err := cfg.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})
}

func TestConfig_ToOptions(t *testing.T) {
	cfg := codex.Config{
		Model:        "gpt-5-codex",
		Timeout:      10 * time.Minute,
		WorkDir:      "/project",
		SandboxMode:  codex.SandboxWorkspaceWrite,
		ApprovalMode: codex.ApprovalNever,
		FullAuto:     true,
		SessionID:    "test-session",
		EnableSearch: true,
		AddDirs:      []string{"/tmp"},
		Images:       []string{"/path/to/img.png"},
		CodexPath:    "/custom/codex",
		Env:          map[string]string{"KEY": "value"},
	}

	opts := cfg.ToOptions()
	assert.NotEmpty(t, opts)

	// Verify options can be applied
	client := codex.NewCodexCLI(opts...)
	assert.NotNil(t, client)
}

func TestConfig_LoadFromEnv(t *testing.T) {
	// Save and restore environment
	t.Setenv("CODEX_MODEL", "test-model")
	t.Setenv("CODEX_TIMEOUT", "10m")
	t.Setenv("CODEX_WORK_DIR", "/test/dir")
	t.Setenv("CODEX_SANDBOX_MODE", "read-only")
	t.Setenv("CODEX_APPROVAL_MODE", "never")
	t.Setenv("CODEX_FULL_AUTO", "true")
	t.Setenv("CODEX_SESSION_ID", "env-session")
	t.Setenv("CODEX_SEARCH", "true")
	t.Setenv("CODEX_PATH", "/env/codex")

	cfg := codex.Config{}
	cfg.LoadFromEnv()

	assert.Equal(t, "test-model", cfg.Model)
	assert.Equal(t, 10*time.Minute, cfg.Timeout)
	assert.Equal(t, "/test/dir", cfg.WorkDir)
	assert.Equal(t, codex.SandboxReadOnly, cfg.SandboxMode)
	assert.Equal(t, codex.ApprovalNever, cfg.ApprovalMode)
	assert.True(t, cfg.FullAuto)
	assert.Equal(t, "env-session", cfg.SessionID)
	assert.True(t, cfg.EnableSearch)
	assert.Equal(t, "/env/codex", cfg.CodexPath)
}

func TestFromEnv(t *testing.T) {
	t.Setenv("CODEX_MODEL", "env-model")

	cfg := codex.FromEnv()

	assert.Equal(t, "env-model", cfg.Model)
	// Default values should still be set
	assert.Equal(t, 5*time.Minute, cfg.Timeout)
	assert.Equal(t, codex.SandboxWorkspaceWrite, cfg.SandboxMode)
}

func TestRoleConstants(t *testing.T) {
	assert.Equal(t, codex.Role("user"), codex.RoleUser)
	assert.Equal(t, codex.Role("assistant"), codex.RoleAssistant)
	assert.Equal(t, codex.Role("tool"), codex.RoleTool)
	assert.Equal(t, codex.Role("system"), codex.RoleSystem)
}

// TestProviderRegistration verifies the codex provider is registered.
func TestProviderRegistration(t *testing.T) {
	// Import the package to trigger init()
	// The actual test is that this compiles and runs without panic
	client := codex.NewCodexCLI()
	require.NotNil(t, client)
}
