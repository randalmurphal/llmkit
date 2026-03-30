package llmkit

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxTurns != 10 {
		t.Fatalf("MaxTurns = %d", cfg.MaxTurns)
	}
	if cfg.Timeout != 5*time.Minute {
		t.Fatalf("Timeout = %v", cfg.Timeout)
	}
}

func TestConfigLoadFromEnv(t *testing.T) {
	t.Setenv("LLMKIT_PROVIDER", "codex")
	t.Setenv("LLMKIT_MODEL", "gpt-5-codex")
	t.Setenv("LLMKIT_MAX_TURNS", "7")
	t.Setenv("LLMKIT_TIMEOUT", "2m")
	t.Setenv("LLMKIT_MAX_BUDGET_USD", "3.5")
	t.Setenv("LLMKIT_WORK_DIR", os.TempDir())

	cfg := Config{}
	cfg.LoadFromEnv()

	if cfg.Provider != "codex" || cfg.Model != "gpt-5-codex" || cfg.MaxTurns != 7 {
		t.Fatalf("unexpected config after env load: %#v", cfg)
	}
	if cfg.Timeout != 2*time.Minute {
		t.Fatalf("Timeout = %v", cfg.Timeout)
	}
}
