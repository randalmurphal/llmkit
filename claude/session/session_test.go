package session

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSessionConfig_Defaults(t *testing.T) {
	cfg := defaultConfig()

	if cfg.claudePath != "claude" {
		t.Errorf("expected claudePath 'claude', got %q", cfg.claudePath)
	}
	if !cfg.dangerouslySkipPermissions {
		t.Error("expected dangerouslySkipPermissions to be true by default")
	}
	if cfg.startupTimeout != 30*time.Second {
		t.Errorf("expected startupTimeout 30s, got %v", cfg.startupTimeout)
	}
	if cfg.idleTimeout != 10*time.Minute {
		t.Errorf("expected idleTimeout 10m, got %v", cfg.idleTimeout)
	}
}

func TestSessionOptions(t *testing.T) {
	tests := []struct {
		name     string
		opt      SessionOption
		validate func(*testing.T, *sessionConfig)
	}{
		{
			name: "WithClaudePath",
			opt:  WithClaudePath("/custom/claude"),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.claudePath != "/custom/claude" {
					t.Errorf("expected '/custom/claude', got %q", c.claudePath)
				}
			},
		},
		{
			name: "WithModel",
			opt:  WithModel("sonnet"),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.model != "sonnet" {
					t.Errorf("expected 'sonnet', got %q", c.model)
				}
			},
		},
		{
			name: "WithWorkdir",
			opt:  WithWorkdir("/test/dir"),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.workdir != "/test/dir" {
					t.Errorf("expected '/test/dir', got %q", c.workdir)
				}
			},
		},
		{
			name: "WithSessionID",
			opt:  WithSessionID("custom-id"),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.sessionID != "custom-id" {
					t.Errorf("expected 'custom-id', got %q", c.sessionID)
				}
			},
		},
		{
			name: "WithResume",
			opt:  WithResume("resume-id"),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.sessionID != "resume-id" {
					t.Errorf("expected sessionID 'resume-id', got %q", c.sessionID)
				}
				if !c.resume {
					t.Error("expected resume to be true")
				}
			},
		},
		{
			name: "WithNoSessionPersistence",
			opt:  WithNoSessionPersistence(),
			validate: func(t *testing.T, c *sessionConfig) {
				if !c.noSessionPersistence {
					t.Error("expected noSessionPersistence to be true")
				}
			},
		},
		{
			name: "WithAllowedTools",
			opt:  WithAllowedTools([]string{"Read", "Write"}),
			validate: func(t *testing.T, c *sessionConfig) {
				if len(c.allowedTools) != 2 {
					t.Errorf("expected 2 tools, got %d", len(c.allowedTools))
				}
			},
		},
		{
			name: "WithDisallowedTools",
			opt:  WithDisallowedTools([]string{"Bash"}),
			validate: func(t *testing.T, c *sessionConfig) {
				if len(c.disallowedTools) != 1 {
					t.Errorf("expected 1 tool, got %d", len(c.disallowedTools))
				}
			},
		},
		{
			name: "WithTools",
			opt:  WithTools([]string{"Read"}),
			validate: func(t *testing.T, c *sessionConfig) {
				if len(c.tools) != 1 {
					t.Errorf("expected 1 tool, got %d", len(c.tools))
				}
			},
		},
		{
			name: "WithPermissions",
			opt:  WithPermissions(false),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.dangerouslySkipPermissions {
					t.Error("expected dangerouslySkipPermissions to be false")
				}
			},
		},
		{
			name: "WithSystemPrompt",
			opt:  WithSystemPrompt("You are a test assistant"),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.systemPrompt != "You are a test assistant" {
					t.Errorf("expected system prompt, got %q", c.systemPrompt)
				}
			},
		},
		{
			name: "WithMaxBudgetUSD",
			opt:  WithMaxBudgetUSD(1.50),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.maxBudgetUSD != 1.50 {
					t.Errorf("expected 1.50, got %f", c.maxBudgetUSD)
				}
			},
		},
		{
			name: "WithMaxTurns",
			opt:  WithMaxTurns(5),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.maxTurns != 5 {
					t.Errorf("expected 5, got %d", c.maxTurns)
				}
			},
		},
		{
			name: "WithStartupTimeout",
			opt:  WithStartupTimeout(1 * time.Minute),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.startupTimeout != 1*time.Minute {
					t.Errorf("expected 1m, got %v", c.startupTimeout)
				}
			},
		},
		{
			name: "WithIdleTimeout",
			opt:  WithIdleTimeout(5 * time.Minute),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.idleTimeout != 5*time.Minute {
					t.Errorf("expected 5m, got %v", c.idleTimeout)
				}
			},
		},
		{
			name: "WithHomeDir",
			opt:  WithHomeDir("/custom/home"),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.homeDir != "/custom/home" {
					t.Errorf("expected '/custom/home', got %q", c.homeDir)
				}
			},
		},
		{
			name: "WithConfigDir",
			opt:  WithConfigDir("/custom/config"),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.configDir != "/custom/config" {
					t.Errorf("expected '/custom/config', got %q", c.configDir)
				}
			},
		},
		{
			name: "WithEnv",
			opt:  WithEnv(map[string]string{"FOO": "bar"}),
			validate: func(t *testing.T, c *sessionConfig) {
				if c.extraEnv["FOO"] != "bar" {
					t.Errorf("expected FOO=bar, got %q", c.extraEnv["FOO"])
				}
			},
		},
		{
			name: "WithIncludeHookOutput",
			opt:  WithIncludeHookOutput(true),
			validate: func(t *testing.T, c *sessionConfig) {
				if !c.includeHookOutput {
					t.Error("expected includeHookOutput to be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			tt.opt(&cfg)
			tt.validate(t, &cfg)
		})
	}
}

func TestSession_BuildArgs(t *testing.T) {
	cfg := sessionConfig{
		model:                      "sonnet",
		sessionID:                  "test-123",
		systemPrompt:               "Be helpful",
		allowedTools:               []string{"Read"},
		disallowedTools:            []string{"Bash"},
		dangerouslySkipPermissions: true,
		maxBudgetUSD:               1.0,
		maxTurns:                   5,
	}

	s := &session{config: cfg}
	args := s.buildArgs()

	// Check required stream-json args
	assertContains(t, args, "--input-format")
	assertContains(t, args, "stream-json")
	assertContains(t, args, "--output-format")
	assertContains(t, args, "--verbose")

	// Check model
	assertContains(t, args, "--model")
	assertContains(t, args, "sonnet")

	// Check session ID
	assertContains(t, args, "--session-id")
	assertContains(t, args, "test-123")

	// Check permissions
	assertContains(t, args, "--dangerously-skip-permissions")

	// Check tools
	assertContains(t, args, "--allowedTools")
	assertContains(t, args, "--disallowedTools")

	// Check limits
	assertContains(t, args, "--max-budget-usd")
	assertContains(t, args, "--max-turns")
	assertContains(t, args, "5")
}

func TestSession_BuildArgs_Resume(t *testing.T) {
	cfg := sessionConfig{
		sessionID: "resume-123",
		resume:    true,
	}

	s := &session{config: cfg}
	args := s.buildArgs()

	assertContains(t, args, "--resume")
	assertContains(t, args, "resume-123")
	assertNotContains(t, args, "--session-id")
}

func TestMockSession_SendAndOutput(t *testing.T) {
	sess := newMockSession("test-123")
	defer func() { _ = sess.Close() }()

	ctx := context.Background()

	// Send a message
	msg := NewUserMessage("Hello!")
	if err := sess.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Verify message was captured
	if len(sess.sendMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(sess.sendMessages))
	}
	if sess.sendMessages[0].Message.Content != "Hello!" {
		t.Errorf("expected 'Hello!', got %q", sess.sendMessages[0].Message.Content)
	}

	// Simulate output
	sess.simulateOutput(OutputMessage{
		Type:      "assistant",
		SessionID: "test-123",
		Assistant: &AssistantMessage{
			Message: ClaudeMessage{
				Content: []ContentBlock{{Type: "text", Text: "Hi there!"}},
			},
		},
	})

	// Read from output channel
	select {
	case out := <-sess.Output():
		if !out.IsAssistant() {
			t.Error("expected assistant message")
		}
		if out.GetText() != "Hi there!" {
			t.Errorf("expected 'Hi there!', got %q", out.GetText())
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for output")
	}
}

func TestMockSession_Close(t *testing.T) {
	sess := newMockSession("test-123")

	if sess.Status() != StatusActive {
		t.Errorf("expected active, got %v", sess.Status())
	}

	if err := sess.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if sess.Status() != StatusClosed {
		t.Errorf("expected closed, got %v", sess.Status())
	}

	// Send should fail after close
	if err := sess.Send(context.Background(), NewUserMessage("test")); err == nil {
		t.Error("expected error on send after close")
	}

	// Double close should be fine
	if err := sess.Close(); err != nil {
		t.Errorf("double close should not error: %v", err)
	}
}

func TestMockSession_Info(t *testing.T) {
	sess := newMockSession("test-123")
	info := sess.Info()

	if info.ID != "test-123" {
		t.Errorf("expected ID 'test-123', got %q", info.ID)
	}
	if info.Status != StatusActive {
		t.Errorf("expected active, got %v", info.Status)
	}
	if info.Model == "" {
		t.Error("expected model to be set")
	}
	if info.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestUserMessage_Marshal(t *testing.T) {
	msg := NewUserMessage("Hello, Claude!")

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := `{"type":"user","message":{"role":"user","content":"Hello, Claude!"}}`
	if string(data) != expected {
		t.Errorf("expected %q, got %q", expected, string(data))
	}
}

// Helper functions

func assertContains(t *testing.T, slice []string, item string) {
	t.Helper()
	for _, s := range slice {
		if s == item {
			return
		}
	}
	t.Errorf("slice %v does not contain %q", slice, item)
}

func assertNotContains(t *testing.T, slice []string, item string) {
	t.Helper()
	for _, s := range slice {
		if s == item {
			t.Errorf("slice should not contain %q", item)
			return
		}
	}
}
