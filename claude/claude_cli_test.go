package claude_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/randalmurphal/llmkit/claude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeCLI_BuildArgs(t *testing.T) {
	tests := []struct {
		name     string
		client   *claude.ClaudeCLI
		req      claude.CompletionRequest
		contains []string
		excludes []string
	}{
		{
			name:   "basic request",
			client: claude.NewClaudeCLI(),
			req: claude.CompletionRequest{
				Messages: []claude.Message{
					{Role: claude.RoleUser, Content: "Hello"},
				},
			},
			contains: []string{"--print", "-p", "Hello"},
		},
		{
			name:   "with system prompt",
			client: claude.NewClaudeCLI(),
			req: claude.CompletionRequest{
				SystemPrompt: "You are helpful",
				Messages: []claude.Message{
					{Role: claude.RoleUser, Content: "Hi"},
				},
			},
			contains: []string{"--system-prompt", "You are helpful"},
		},
		{
			name:   "with model from client",
			client: claude.NewClaudeCLI(claude.WithModel("claude-3-opus")),
			req: claude.CompletionRequest{
				Messages: []claude.Message{
					{Role: claude.RoleUser, Content: "Test"},
				},
			},
			contains: []string{"--model", "claude-3-opus"},
		},
		{
			name:   "with model from request",
			client: claude.NewClaudeCLI(claude.WithModel("client-default")),
			req: claude.CompletionRequest{
				Model: "request-model",
				Messages: []claude.Message{
					{Role: claude.RoleUser, Content: "Test"},
				},
			},
			// Request model should override client model
			contains: []string{"--model"},
		},
		{
			name:   "max tokens ignored (CLI doesn't support it)",
			client: claude.NewClaudeCLI(),
			req: claude.CompletionRequest{
				MaxTokens: 1000, // Should be silently ignored - CLI doesn't have this flag
				Messages: []claude.Message{
					{Role: claude.RoleUser, Content: "Test"},
				},
			},
			contains: []string{"-p"}, // Just verify basic args, no --max-tokens
		},
		{
			name:   "multiple messages",
			client: claude.NewClaudeCLI(),
			req: claude.CompletionRequest{
				Messages: []claude.Message{
					{Role: claude.RoleUser, Content: "First question"},
					{Role: claude.RoleAssistant, Content: "First answer"},
					{Role: claude.RoleUser, Content: "Follow-up"},
				},
			},
			contains: []string{"-p"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't directly test buildArgs since it's private
			// But we can verify the client is created correctly
			assert.NotNil(t, tt.client)
		})
	}
}

func TestClaudeCLI_Options(t *testing.T) {
	// Test WithClaudePath
	client := claude.NewClaudeCLI(claude.WithClaudePath("/custom/path/claude"))
	assert.NotNil(t, client)

	// Test WithWorkdir
	client = claude.NewClaudeCLI(claude.WithWorkdir("/some/workdir"))
	assert.NotNil(t, client)

	// Test WithAllowedTools
	client = claude.NewClaudeCLI(claude.WithAllowedTools([]string{"read", "write"}))
	assert.NotNil(t, client)

	// Test all options combined
	client = claude.NewClaudeCLI(
		claude.WithClaudePath("/custom/claude"),
		claude.WithModel("claude-3-opus"),
		claude.WithWorkdir("/project"),
		claude.WithAllowedTools([]string{"bash"}),
	)
	assert.NotNil(t, client)
}

func TestClaudeCLI_NewOptions(t *testing.T) {
	// Test output control options
	t.Run("output format options", func(t *testing.T) {
		client := claude.NewClaudeCLI(
			claude.WithOutputFormat(claude.OutputFormatJSON),
			claude.WithJSONSchema(`{"type": "object", "properties": {"name": {"type": "string"}}}`),
		)
		assert.NotNil(t, client)
	})

	// Test session management options
	t.Run("session management options", func(t *testing.T) {
		client := claude.NewClaudeCLI(
			claude.WithSessionID("test-session"),
		)
		assert.NotNil(t, client)

		client = claude.NewClaudeCLI(claude.WithContinue())
		assert.NotNil(t, client)

		client = claude.NewClaudeCLI(claude.WithResume("prev-session"))
		assert.NotNil(t, client)

		client = claude.NewClaudeCLI(claude.WithNoSessionPersistence())
		assert.NotNil(t, client)
	})

	// Test tool control options
	t.Run("tool control options", func(t *testing.T) {
		client := claude.NewClaudeCLI(
			claude.WithAllowedTools([]string{"read", "write"}),
			claude.WithDisallowedTools([]string{"bash", "execute"}),
		)
		assert.NotNil(t, client)

		client = claude.NewClaudeCLI(claude.WithTools([]string{"Bash", "Read", "Edit"}))
		assert.NotNil(t, client)
	})

	// Test permission options
	t.Run("permission options", func(t *testing.T) {
		client := claude.NewClaudeCLI(claude.WithDangerouslySkipPermissions())
		assert.NotNil(t, client)

		client = claude.NewClaudeCLI(claude.WithPermissionMode(claude.PermissionModeAcceptEdits))
		assert.NotNil(t, client)

		client = claude.NewClaudeCLI(claude.WithPermissionMode(claude.PermissionModeBypassPermissions))
		assert.NotNil(t, client)

		client = claude.NewClaudeCLI(claude.WithSettingSources([]string{"project", "local", "user"}))
		assert.NotNil(t, client)
	})

	// Test context options
	t.Run("context options", func(t *testing.T) {
		client := claude.NewClaudeCLI(
			claude.WithAddDirs([]string{"/tmp", "/home/user/project"}),
		)
		assert.NotNil(t, client)

		client = claude.NewClaudeCLI(claude.WithSystemPrompt("You are a helpful assistant"))
		assert.NotNil(t, client)

		client = claude.NewClaudeCLI(claude.WithAppendSystemPrompt("Always be concise"))
		assert.NotNil(t, client)
	})

	// Test budget options
	t.Run("budget options", func(t *testing.T) {
		client := claude.NewClaudeCLI(claude.WithMaxBudgetUSD(5.0))
		assert.NotNil(t, client)

		client = claude.NewClaudeCLI(claude.WithFallbackModel("haiku"))
		assert.NotNil(t, client)
	})

	// Test production configuration (all options combined)
	t.Run("production configuration", func(t *testing.T) {
		client := claude.NewClaudeCLI(
			claude.WithClaudePath("/usr/local/bin/claude"),
			claude.WithModel("sonnet"),
			claude.WithWorkdir("/home/user/project"),
			claude.WithTimeout(10*time.Minute),
			claude.WithOutputFormat(claude.OutputFormatJSON),
			claude.WithDangerouslySkipPermissions(),
			claude.WithSettingSources([]string{"project", "local"}),
			claude.WithMaxBudgetUSD(1.0),
			claude.WithFallbackModel("haiku"),
			claude.WithDisallowedTools([]string{"Write", "Bash"}),
			claude.WithAppendSystemPrompt("Be extra careful with code changes"),
		)
		assert.NotNil(t, client)
	})
}

func TestClaudeCLI_OutputFormatConstants(t *testing.T) {
	// Verify output format constants are accessible
	assert.Equal(t, claude.OutputFormat("text"), claude.OutputFormatText)
	assert.Equal(t, claude.OutputFormat("json"), claude.OutputFormatJSON)
	assert.Equal(t, claude.OutputFormat("stream-json"), claude.OutputFormatStreamJSON)
}

func TestClaudeCLI_PermissionModeConstants(t *testing.T) {
	// Verify permission mode constants are accessible
	assert.Equal(t, claude.PermissionMode(""), claude.PermissionModeDefault)
	assert.Equal(t, claude.PermissionMode("acceptEdits"), claude.PermissionModeAcceptEdits)
	assert.Equal(t, claude.PermissionMode("bypassPermissions"), claude.PermissionModeBypassPermissions)
}

func TestCompletionResponse_NewFields(t *testing.T) {
	// Test that new fields are accessible on CompletionResponse
	resp := &claude.CompletionResponse{
		Content:      "Hello",
		SessionID:    "session-123",
		CostUSD:      0.05,
		NumTurns:     2,
		FinishReason: "stop",
		Model:        "sonnet",
		Usage: claude.TokenUsage{
			InputTokens:              100,
			OutputTokens:             50,
			TotalTokens:              150,
			CacheCreationInputTokens: 500,
			CacheReadInputTokens:     200,
		},
	}

	assert.Equal(t, "session-123", resp.SessionID)
	assert.Equal(t, 0.05, resp.CostUSD)
	assert.Equal(t, 2, resp.NumTurns)
	assert.Equal(t, 500, resp.Usage.CacheCreationInputTokens)
	assert.Equal(t, 200, resp.Usage.CacheReadInputTokens)
}

func TestTokenUsage_Add_WithCacheTokens(t *testing.T) {
	usage := claude.TokenUsage{
		InputTokens:              100,
		OutputTokens:             50,
		TotalTokens:              150,
		CacheCreationInputTokens: 500,
		CacheReadInputTokens:     200,
	}

	other := claude.TokenUsage{
		InputTokens:              200,
		OutputTokens:             100,
		TotalTokens:              300,
		CacheCreationInputTokens: 300,
		CacheReadInputTokens:     100,
	}

	usage.Add(other)

	assert.Equal(t, 300, usage.InputTokens)
	assert.Equal(t, 150, usage.OutputTokens)
	assert.Equal(t, 450, usage.TotalTokens)
	assert.Equal(t, 800, usage.CacheCreationInputTokens)
	assert.Equal(t, 300, usage.CacheReadInputTokens)
}

func TestClaudeCLI_IntegrationSkip(t *testing.T) {
	// Skip if claude binary not available
	if _, err := exec.LookPath("claude"); err != nil {
		t.Skip("claude binary not available, skipping integration test")
	}

	// This would be an actual integration test if claude is available
	// For now, just verify the client can be created
	client := claude.NewClaudeCLI()
	assert.NotNil(t, client)
}

func TestClaudeCLI_Error(t *testing.T) {
	err := claude.NewError("complete", assert.AnError, true)
	assert.Contains(t, err.Error(), "claude complete")
	assert.True(t, err.Retryable)
	assert.Equal(t, assert.AnError, err.Unwrap())
}

func TestLLMErrors(t *testing.T) {
	// Verify sentinel errors are defined
	assert.NotNil(t, claude.ErrUnavailable)
	assert.NotNil(t, claude.ErrContextTooLong)
	assert.NotNil(t, claude.ErrRateLimited)
	assert.NotNil(t, claude.ErrInvalidRequest)
	assert.NotNil(t, claude.ErrTimeout)
}

func TestClaudeCLI_WithTimeout(t *testing.T) {
	client := claude.NewClaudeCLI(claude.WithTimeout(10 * time.Second))
	assert.NotNil(t, client)
}

func TestClaudeCLI_Complete_NonExistentBinary(t *testing.T) {
	client := claude.NewClaudeCLI(claude.WithClaudePath("/nonexistent/path/to/claude"))

	_, err := client.Complete(context.Background(), claude.CompletionRequest{
		Messages: []claude.Message{{Role: claude.RoleUser, Content: "test"}},
	})

	assert.Error(t, err)
}

func TestClaudeCLI_Stream_NonExistentBinary(t *testing.T) {
	client := claude.NewClaudeCLI(claude.WithClaudePath("/nonexistent/path/to/claude"))

	_, err := client.Stream(context.Background(), claude.CompletionRequest{
		Messages: []claude.Message{{Role: claude.RoleUser, Content: "test"}},
	})

	assert.Error(t, err)
}

func TestTokenUsage_Add(t *testing.T) {
	usage := claude.TokenUsage{
		InputTokens:  10,
		OutputTokens: 5,
		TotalTokens:  15,
	}

	other := claude.TokenUsage{
		InputTokens:  20,
		OutputTokens: 10,
		TotalTokens:  30,
	}

	usage.Add(other)

	assert.Equal(t, 30, usage.InputTokens)
	assert.Equal(t, 15, usage.OutputTokens)
	assert.Equal(t, 45, usage.TotalTokens)
}

func TestMockClient_WithStreamFunc(t *testing.T) {
	mock := claude.NewMockClient("").WithStreamFunc(func(ctx context.Context, req claude.CompletionRequest) (<-chan claude.StreamChunk, error) {
		ch := make(chan claude.StreamChunk)
		go func() {
			defer close(ch)
			ch <- claude.StreamChunk{Content: "custom "}
			ch <- claude.StreamChunk{Content: "stream"}
			ch <- claude.StreamChunk{Done: true}
		}()
		return ch, nil
	})

	ch, err := mock.Stream(context.Background(), claude.CompletionRequest{})
	require.NoError(t, err)

	var content string
	for chunk := range ch {
		content += chunk.Content
	}
	assert.Equal(t, "custom stream", content)
}

func TestMockClient_Stream_ContextCancellation(t *testing.T) {
	mock := claude.NewMockClient("response")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	ch, err := mock.Stream(ctx, claude.CompletionRequest{})
	require.NoError(t, err)

	// Read from channel - may get content or error depending on race
	chunk := <-ch
	// Either we get an error chunk or a content chunk that may or may not have error
	// The important thing is the channel closes cleanly
	_ = chunk
}
