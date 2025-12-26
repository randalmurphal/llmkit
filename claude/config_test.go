package claude

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want %q", cfg.Model, "claude-sonnet-4-20250514")
	}
	if cfg.MaxTurns != 10 {
		t.Errorf("MaxTurns = %d, want %d", cfg.MaxTurns, 10)
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 5*time.Minute)
	}
	if cfg.OutputFormat != OutputFormatJSON {
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, OutputFormatJSON)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name:    "valid minimal config",
			cfg:     Config{Model: "claude-sonnet-4-20250514"},
			wantErr: false,
		},
		{
			name:    "empty model",
			cfg:     Config{},
			wantErr: true,
		},
		{
			name:    "negative max turns",
			cfg:     Config{Model: "claude-sonnet-4-20250514", MaxTurns: -1},
			wantErr: true,
		},
		{
			name:    "negative max budget",
			cfg:     Config{Model: "claude-sonnet-4-20250514", MaxBudgetUSD: -1},
			wantErr: true,
		},
		{
			name:    "negative timeout",
			cfg:     Config{Model: "claude-sonnet-4-20250514", Timeout: -1},
			wantErr: true,
		},
		{
			name: "full valid config",
			cfg: Config{
				Model:                      "claude-opus-4-5-20251101",
				FallbackModel:              "claude-sonnet-4-20250514",
				SystemPrompt:               "You are helpful.",
				MaxTurns:                   20,
				Timeout:                    10 * time.Minute,
				MaxBudgetUSD:               5.0,
				WorkDir:                    "/app",
				AllowedTools:               []string{"Read", "Bash"},
				DangerouslySkipPermissions: true,
				HomeDir:                    "/home/claude",
				OutputFormat:               OutputFormatJSON,
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

func TestConfig_LoadFromEnv(t *testing.T) {
	// Save original env and restore after test
	originalEnv := map[string]string{
		"CLAUDE_MODEL":                  os.Getenv("CLAUDE_MODEL"),
		"CLAUDE_FALLBACK_MODEL":         os.Getenv("CLAUDE_FALLBACK_MODEL"),
		"CLAUDE_SYSTEM_PROMPT":          os.Getenv("CLAUDE_SYSTEM_PROMPT"),
		"CLAUDE_APPEND_SYSTEM_PROMPT":   os.Getenv("CLAUDE_APPEND_SYSTEM_PROMPT"),
		"CLAUDE_MAX_TURNS":              os.Getenv("CLAUDE_MAX_TURNS"),
		"CLAUDE_TIMEOUT":                os.Getenv("CLAUDE_TIMEOUT"),
		"CLAUDE_MAX_BUDGET_USD":         os.Getenv("CLAUDE_MAX_BUDGET_USD"),
		"CLAUDE_WORK_DIR":               os.Getenv("CLAUDE_WORK_DIR"),
		"CLAUDE_PATH":                   os.Getenv("CLAUDE_PATH"),
		"CLAUDE_HOME_DIR":               os.Getenv("CLAUDE_HOME_DIR"),
		"CLAUDE_CONFIG_DIR":             os.Getenv("CLAUDE_CONFIG_DIR"),
		"CLAUDE_OUTPUT_FORMAT":          os.Getenv("CLAUDE_OUTPUT_FORMAT"),
		"CLAUDE_SKIP_PERMISSIONS":       os.Getenv("CLAUDE_SKIP_PERMISSIONS"),
		"CLAUDE_PERMISSION_MODE":        os.Getenv("CLAUDE_PERMISSION_MODE"),
		"CLAUDE_SESSION_ID":             os.Getenv("CLAUDE_SESSION_ID"),
		"CLAUDE_NO_SESSION_PERSISTENCE": os.Getenv("CLAUDE_NO_SESSION_PERSISTENCE"),
	}
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Clear env vars
	for k := range originalEnv {
		os.Unsetenv(k)
	}

	// Set test values
	os.Setenv("CLAUDE_MODEL", "claude-opus-4-5-20251101")
	os.Setenv("CLAUDE_FALLBACK_MODEL", "claude-haiku-3-5-20250626")
	os.Setenv("CLAUDE_SYSTEM_PROMPT", "Test prompt")
	os.Setenv("CLAUDE_APPEND_SYSTEM_PROMPT", "Appended text")
	os.Setenv("CLAUDE_MAX_TURNS", "25")
	os.Setenv("CLAUDE_TIMEOUT", "10m")
	os.Setenv("CLAUDE_MAX_BUDGET_USD", "15.5")
	os.Setenv("CLAUDE_WORK_DIR", "/custom/dir")
	os.Setenv("CLAUDE_PATH", "/usr/local/bin/claude")
	os.Setenv("CLAUDE_HOME_DIR", "/home/test")
	os.Setenv("CLAUDE_CONFIG_DIR", "/config/.claude")
	os.Setenv("CLAUDE_OUTPUT_FORMAT", "stream-json")
	os.Setenv("CLAUDE_SKIP_PERMISSIONS", "true")
	os.Setenv("CLAUDE_PERMISSION_MODE", "acceptEdits")
	os.Setenv("CLAUDE_SESSION_ID", "session-123")
	os.Setenv("CLAUDE_NO_SESSION_PERSISTENCE", "1")

	cfg := Config{}
	cfg.LoadFromEnv()

	if cfg.Model != "claude-opus-4-5-20251101" {
		t.Errorf("Model = %q, want %q", cfg.Model, "claude-opus-4-5-20251101")
	}
	if cfg.FallbackModel != "claude-haiku-3-5-20250626" {
		t.Errorf("FallbackModel = %q, want %q", cfg.FallbackModel, "claude-haiku-3-5-20250626")
	}
	if cfg.SystemPrompt != "Test prompt" {
		t.Errorf("SystemPrompt = %q, want %q", cfg.SystemPrompt, "Test prompt")
	}
	if cfg.AppendSystemPrompt != "Appended text" {
		t.Errorf("AppendSystemPrompt = %q, want %q", cfg.AppendSystemPrompt, "Appended text")
	}
	if cfg.MaxTurns != 25 {
		t.Errorf("MaxTurns = %d, want %d", cfg.MaxTurns, 25)
	}
	if cfg.Timeout != 10*time.Minute {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 10*time.Minute)
	}
	if cfg.MaxBudgetUSD != 15.5 {
		t.Errorf("MaxBudgetUSD = %f, want %f", cfg.MaxBudgetUSD, 15.5)
	}
	if cfg.WorkDir != "/custom/dir" {
		t.Errorf("WorkDir = %q, want %q", cfg.WorkDir, "/custom/dir")
	}
	if cfg.ClaudePath != "/usr/local/bin/claude" {
		t.Errorf("ClaudePath = %q, want %q", cfg.ClaudePath, "/usr/local/bin/claude")
	}
	if cfg.HomeDir != "/home/test" {
		t.Errorf("HomeDir = %q, want %q", cfg.HomeDir, "/home/test")
	}
	if cfg.ConfigDir != "/config/.claude" {
		t.Errorf("ConfigDir = %q, want %q", cfg.ConfigDir, "/config/.claude")
	}
	if cfg.OutputFormat != OutputFormatStreamJSON {
		t.Errorf("OutputFormat = %q, want %q", cfg.OutputFormat, OutputFormatStreamJSON)
	}
	if !cfg.DangerouslySkipPermissions {
		t.Error("DangerouslySkipPermissions should be true")
	}
	if cfg.PermissionMode != PermissionModeAcceptEdits {
		t.Errorf("PermissionMode = %q, want %q", cfg.PermissionMode, PermissionModeAcceptEdits)
	}
	if cfg.SessionID != "session-123" {
		t.Errorf("SessionID = %q, want %q", cfg.SessionID, "session-123")
	}
	if !cfg.NoSessionPersistence {
		t.Error("NoSessionPersistence should be true")
	}
}

func TestFromEnv(t *testing.T) {
	// Save and restore env
	originalModel := os.Getenv("CLAUDE_MODEL")
	defer func() {
		if originalModel == "" {
			os.Unsetenv("CLAUDE_MODEL")
		} else {
			os.Setenv("CLAUDE_MODEL", originalModel)
		}
	}()

	os.Setenv("CLAUDE_MODEL", "claude-opus-4-5-20251101")

	cfg := FromEnv()

	// Should have env override
	if cfg.Model != "claude-opus-4-5-20251101" {
		t.Errorf("Model = %q, want %q", cfg.Model, "claude-opus-4-5-20251101")
	}
	// Should have defaults for non-env values
	if cfg.MaxTurns != 10 {
		t.Errorf("MaxTurns = %d, want %d (default)", cfg.MaxTurns, 10)
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want %v (default)", cfg.Timeout, 5*time.Minute)
	}
}

func TestConfig_ToOptions(t *testing.T) {
	cfg := Config{
		Model:                      "claude-opus-4-5-20251101",
		FallbackModel:              "claude-sonnet-4-20250514",
		SystemPrompt:               "Test system prompt",
		AppendSystemPrompt:         "Appended",
		MaxTurns:                   15,
		Timeout:                    3 * time.Minute,
		MaxBudgetUSD:               5.0,
		WorkDir:                    "/test/dir",
		AllowedTools:               []string{"Read", "Bash"},
		DisallowedTools:            []string{"Write"},
		Tools:                      []string{"Read"},
		DangerouslySkipPermissions: true,
		PermissionMode:             PermissionModeAcceptEdits,
		SessionID:                  "test-session",
		Continue:                   true,
		Resume:                     "resume-id",
		NoSessionPersistence:       true,
		HomeDir:                    "/home/test",
		ConfigDir:                  "/config/.claude",
		Env:                        map[string]string{"FOO": "bar"},
		OutputFormat:               OutputFormatJSON,
		JSONSchema:                 `{"type": "object"}`,
		AddDirs:                    []string{"/extra/dir"},
		SettingSources:             []string{"project", "user"},
		ClaudePath:                 "/custom/claude",
	}

	opts := cfg.ToOptions()

	// Verify we got options (exact count depends on implementation)
	if len(opts) == 0 {
		t.Error("ToOptions() returned empty slice")
	}

	// Apply options to a client and verify configuration
	cli := NewClaudeCLI(opts...)

	if cli.model != "claude-opus-4-5-20251101" {
		t.Errorf("model = %q, want %q", cli.model, "claude-opus-4-5-20251101")
	}
	if cli.fallbackModel != "claude-sonnet-4-20250514" {
		t.Errorf("fallbackModel = %q, want %q", cli.fallbackModel, "claude-sonnet-4-20250514")
	}
	if cli.systemPrompt != "Test system prompt" {
		t.Errorf("systemPrompt = %q, want %q", cli.systemPrompt, "Test system prompt")
	}
	if cli.maxTurns != 15 {
		t.Errorf("maxTurns = %d, want %d", cli.maxTurns, 15)
	}
	if cli.timeout != 3*time.Minute {
		t.Errorf("timeout = %v, want %v", cli.timeout, 3*time.Minute)
	}
	if cli.workdir != "/test/dir" {
		t.Errorf("workdir = %q, want %q", cli.workdir, "/test/dir")
	}
	if !cli.dangerouslySkipPermissions {
		t.Error("dangerouslySkipPermissions should be true")
	}
	if cli.homeDir != "/home/test" {
		t.Errorf("homeDir = %q, want %q", cli.homeDir, "/home/test")
	}
}

func TestNewFromConfig(t *testing.T) {
	cfg := Config{
		Model:    "claude-opus-4-5-20251101",
		MaxTurns: 20,
	}

	client := NewFromConfig(cfg)

	// Verify it's a valid client
	if client == nil {
		t.Fatal("NewFromConfig returned nil")
	}

	// Verify type
	_, ok := client.(*ClaudeCLI)
	if !ok {
		t.Error("NewFromConfig should return *ClaudeCLI")
	}
}

func TestNewFromConfig_WithOptionOverrides(t *testing.T) {
	cfg := Config{
		Model:    "claude-sonnet-4-20250514",
		MaxTurns: 10,
	}

	// Override model with option
	client := NewFromConfig(cfg, WithModel("claude-opus-4-5-20251101"))

	cli := client.(*ClaudeCLI)
	// Option should override config
	if cli.model != "claude-opus-4-5-20251101" {
		t.Errorf("model = %q, want %q (option should override)", cli.model, "claude-opus-4-5-20251101")
	}
}

func TestSingleton(t *testing.T) {
	// Reset singleton state
	ResetDefaultClient()

	// Set config before getting client
	SetDefaultConfig(Config{
		Model:    "claude-opus-4-5-20251101",
		MaxTurns: 5,
	})

	// Get client
	client1 := GetDefaultClient()
	if client1 == nil {
		t.Fatal("GetDefaultClient returned nil")
	}

	// Get again - should be same instance
	client2 := GetDefaultClient()
	if client1 != client2 {
		t.Error("GetDefaultClient should return same instance")
	}

	// Verify config was applied
	cli := client1.(*ClaudeCLI)
	if cli.model != "claude-opus-4-5-20251101" {
		t.Errorf("model = %q, want %q", cli.model, "claude-opus-4-5-20251101")
	}

	// Clean up
	ResetDefaultClient()
}

func TestSetDefaultClient(t *testing.T) {
	ResetDefaultClient()

	mock := NewMockClient("test response")
	SetDefaultClient(mock)

	client := GetDefaultClient()
	// Compare by doing a type assertion to the same type
	mockClient, ok := client.(*MockClient)
	if !ok || mockClient != mock {
		t.Error("GetDefaultClient should return the mock we set")
	}

	// Clean up
	ResetDefaultClient()
}

func TestResetDefaultClient(t *testing.T) {
	mock := NewMockClient("test")
	SetDefaultClient(mock)

	ResetDefaultClient()

	// After reset, should create a new client
	client := GetDefaultClient()
	// Compare by doing a type assertion - after reset it should be *ClaudeCLI not *MockClient
	_, isMock := client.(*MockClient)
	if isMock {
		t.Error("After reset, GetDefaultClient should return new client, not the mock")
	}

	// Clean up
	ResetDefaultClient()
}

func TestContextWithClient(t *testing.T) {
	mock := NewMockClient("test")
	ctx := ContextWithClient(context.Background(), mock)

	retrieved := ClientFromContext(ctx)
	if retrieved == nil {
		t.Fatal("ClientFromContext returned nil")
	}
	mockClient, ok := retrieved.(*MockClient)
	if !ok || mockClient != mock {
		t.Error("ClientFromContext returned different client")
	}
}

func TestClientFromContext_Missing(t *testing.T) {
	ctx := context.Background()

	client := ClientFromContext(ctx)
	if client != nil {
		t.Errorf("expected nil, got %v", client)
	}
}

func TestMustClientFromContext_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, got none")
		}
	}()

	ctx := context.Background()
	MustClientFromContext(ctx)
}

func TestMustClientFromContext_Success(t *testing.T) {
	mock := NewMockClient("test")
	ctx := ContextWithClient(context.Background(), mock)

	// Should not panic
	retrieved := MustClientFromContext(ctx)
	mockClient, ok := retrieved.(*MockClient)
	if !ok || mockClient != mock {
		t.Error("MustClientFromContext returned different client")
	}
}

func TestConfig_EnvInvalidValues(t *testing.T) {
	// Save original env
	originalMaxTurns := os.Getenv("CLAUDE_MAX_TURNS")
	originalTimeout := os.Getenv("CLAUDE_TIMEOUT")
	originalBudget := os.Getenv("CLAUDE_MAX_BUDGET_USD")
	defer func() {
		if originalMaxTurns == "" {
			os.Unsetenv("CLAUDE_MAX_TURNS")
		} else {
			os.Setenv("CLAUDE_MAX_TURNS", originalMaxTurns)
		}
		if originalTimeout == "" {
			os.Unsetenv("CLAUDE_TIMEOUT")
		} else {
			os.Setenv("CLAUDE_TIMEOUT", originalTimeout)
		}
		if originalBudget == "" {
			os.Unsetenv("CLAUDE_MAX_BUDGET_USD")
		} else {
			os.Setenv("CLAUDE_MAX_BUDGET_USD", originalBudget)
		}
	}()

	// Set invalid values
	os.Setenv("CLAUDE_MAX_TURNS", "not-a-number")
	os.Setenv("CLAUDE_TIMEOUT", "invalid")
	os.Setenv("CLAUDE_MAX_BUDGET_USD", "not-a-float")

	cfg := Config{
		MaxTurns:     5,
		Timeout:      1 * time.Minute,
		MaxBudgetUSD: 10.0,
	}
	cfg.LoadFromEnv()

	// Invalid env values should be ignored, keeping original values
	if cfg.MaxTurns != 5 {
		t.Errorf("MaxTurns = %d, want %d (invalid env should be ignored)", cfg.MaxTurns, 5)
	}
	if cfg.Timeout != 1*time.Minute {
		t.Errorf("Timeout = %v, want %v (invalid env should be ignored)", cfg.Timeout, 1*time.Minute)
	}
	if cfg.MaxBudgetUSD != 10.0 {
		t.Errorf("MaxBudgetUSD = %f, want %f (invalid env should be ignored)", cfg.MaxBudgetUSD, 10.0)
	}
}
