package provider

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxTurns != 10 {
		t.Errorf("expected MaxTurns=10, got %d", cfg.MaxTurns)
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("expected Timeout=5m, got %v", cfg.Timeout)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     Config{Provider: "test"},
			wantErr: false,
		},
		{
			name:    "missing provider",
			cfg:     Config{},
			wantErr: true,
		},
		{
			name:    "negative max_turns",
			cfg:     Config{Provider: "test", MaxTurns: -1},
			wantErr: true,
		},
		{
			name:    "negative max_budget",
			cfg:     Config{Provider: "test", MaxBudgetUSD: -1},
			wantErr: true,
		},
		{
			name:    "negative timeout",
			cfg:     Config{Provider: "test", Timeout: -1 * time.Second},
			wantErr: true,
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
	// Save and restore environment
	oldEnv := map[string]string{
		"LLMKIT_PROVIDER":       os.Getenv("LLMKIT_PROVIDER"),
		"LLMKIT_MODEL":          os.Getenv("LLMKIT_MODEL"),
		"LLMKIT_MAX_TURNS":      os.Getenv("LLMKIT_MAX_TURNS"),
		"LLMKIT_TIMEOUT":        os.Getenv("LLMKIT_TIMEOUT"),
		"LLMKIT_MAX_BUDGET_USD": os.Getenv("LLMKIT_MAX_BUDGET_USD"),
	}
	defer func() {
		for k, v := range oldEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	// Set test values
	os.Setenv("LLMKIT_PROVIDER", "claude")
	os.Setenv("LLMKIT_MODEL", "test-model")
	os.Setenv("LLMKIT_MAX_TURNS", "20")
	os.Setenv("LLMKIT_TIMEOUT", "10m")
	os.Setenv("LLMKIT_MAX_BUDGET_USD", "5.5")

	cfg := Config{}
	cfg.LoadFromEnv()

	if cfg.Provider != "claude" {
		t.Errorf("expected Provider='claude', got %q", cfg.Provider)
	}
	if cfg.Model != "test-model" {
		t.Errorf("expected Model='test-model', got %q", cfg.Model)
	}
	if cfg.MaxTurns != 20 {
		t.Errorf("expected MaxTurns=20, got %d", cfg.MaxTurns)
	}
	if cfg.Timeout != 10*time.Minute {
		t.Errorf("expected Timeout=10m, got %v", cfg.Timeout)
	}
	if cfg.MaxBudgetUSD != 5.5 {
		t.Errorf("expected MaxBudgetUSD=5.5, got %f", cfg.MaxBudgetUSD)
	}
}

func TestConfig_WithMethods(t *testing.T) {
	cfg := Config{}

	cfg = cfg.WithProvider("claude")
	if cfg.Provider != "claude" {
		t.Errorf("WithProvider failed: expected 'claude', got %q", cfg.Provider)
	}

	cfg = cfg.WithModel("test-model")
	if cfg.Model != "test-model" {
		t.Errorf("WithModel failed: expected 'test-model', got %q", cfg.Model)
	}

	cfg = cfg.WithWorkDir("/tmp")
	if cfg.WorkDir != "/tmp" {
		t.Errorf("WithWorkDir failed: expected '/tmp', got %q", cfg.WorkDir)
	}

	cfg = cfg.WithOption("key", "value")
	if cfg.GetStringOption("key", "") != "value" {
		t.Errorf("WithOption failed: expected 'value', got %q", cfg.GetStringOption("key", ""))
	}
}

func TestConfig_GetOptions(t *testing.T) {
	cfg := Config{
		Options: map[string]any{
			"string_opt": "hello",
			"bool_opt":   true,
			"int_opt":    42,
			"float_opt":  3.14,
		},
	}

	// String option
	if got := cfg.GetStringOption("string_opt", ""); got != "hello" {
		t.Errorf("GetStringOption: expected 'hello', got %q", got)
	}
	if got := cfg.GetStringOption("missing", "default"); got != "default" {
		t.Errorf("GetStringOption: expected 'default', got %q", got)
	}

	// Bool option
	if got := cfg.GetBoolOption("bool_opt", false); !got {
		t.Errorf("GetBoolOption: expected true, got false")
	}
	if got := cfg.GetBoolOption("missing", true); !got {
		t.Errorf("GetBoolOption: expected true (default), got false")
	}

	// Int option
	if got := cfg.GetIntOption("int_opt", 0); got != 42 {
		t.Errorf("GetIntOption: expected 42, got %d", got)
	}
	if got := cfg.GetIntOption("missing", 100); got != 100 {
		t.Errorf("GetIntOption: expected 100 (default), got %d", got)
	}

	// Float to int conversion
	if got := cfg.GetIntOption("float_opt", 0); got != 3 {
		t.Errorf("GetIntOption from float: expected 3, got %d", got)
	}
}
