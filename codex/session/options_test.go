package session

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.codexPath != "codex" {
		t.Errorf("expected codexPath 'codex', got %q", cfg.codexPath)
	}
	if cfg.startupTimeout != 30*time.Second {
		t.Errorf("expected startupTimeout 30s, got %v", cfg.startupTimeout)
	}
	if cfg.idleTimeout != 10*time.Minute {
		t.Errorf("expected idleTimeout 10m, got %v", cfg.idleTimeout)
	}
	if cfg.model != "" {
		t.Errorf("expected empty model, got %q", cfg.model)
	}
	if cfg.workdir != "" {
		t.Errorf("expected empty workdir, got %q", cfg.workdir)
	}
	if cfg.sandboxMode != "" {
		t.Errorf("expected empty sandboxMode, got %q", cfg.sandboxMode)
	}
	if cfg.approvalMode != "" {
		t.Errorf("expected empty approvalMode, got %q", cfg.approvalMode)
	}
	if cfg.fullAuto {
		t.Error("expected fullAuto to be false")
	}
	if cfg.threadID != "" {
		t.Errorf("expected empty threadID, got %q", cfg.threadID)
	}
	if cfg.resume {
		t.Error("expected resume to be false")
	}
	if cfg.systemPrompt != "" {
		t.Errorf("expected empty systemPrompt, got %q", cfg.systemPrompt)
	}
	if cfg.reasoningEffort != "" {
		t.Errorf("expected empty reasoningEffort, got %q", cfg.reasoningEffort)
	}
	if len(cfg.enabledFeatures) != 0 {
		t.Errorf("expected empty enabledFeatures, got %v", cfg.enabledFeatures)
	}
	if len(cfg.disabledFeatures) != 0 {
		t.Errorf("expected empty disabledFeatures, got %v", cfg.disabledFeatures)
	}
	if cfg.extraEnv != nil {
		t.Errorf("expected nil extraEnv, got %v", cfg.extraEnv)
	}
}

func TestWithCodexPath(t *testing.T) {
	cfg := defaultConfig()
	WithCodexPath("/custom/codex")(&cfg)

	if cfg.codexPath != "/custom/codex" {
		t.Errorf("expected '/custom/codex', got %q", cfg.codexPath)
	}
}

func TestWithModel(t *testing.T) {
	cfg := defaultConfig()
	WithModel("o4-mini")(&cfg)

	if cfg.model != "o4-mini" {
		t.Errorf("expected 'o4-mini', got %q", cfg.model)
	}
}

func TestWithWorkdir(t *testing.T) {
	cfg := defaultConfig()
	WithWorkdir("/my/project")(&cfg)

	if cfg.workdir != "/my/project" {
		t.Errorf("expected '/my/project', got %q", cfg.workdir)
	}
}

func TestWithSandboxMode(t *testing.T) {
	cfg := defaultConfig()
	WithSandboxMode("read-only")(&cfg)

	if cfg.sandboxMode != "read-only" {
		t.Errorf("expected 'read-only', got %q", cfg.sandboxMode)
	}
}

func TestWithApprovalMode(t *testing.T) {
	cfg := defaultConfig()
	WithApprovalMode("never")(&cfg)

	if cfg.approvalMode != "never" {
		t.Errorf("expected 'never', got %q", cfg.approvalMode)
	}
}

func TestWithFullAuto(t *testing.T) {
	cfg := defaultConfig()
	WithFullAuto()(&cfg)

	if !cfg.fullAuto {
		t.Error("expected fullAuto to be true")
	}
}

func TestWithThreadID(t *testing.T) {
	cfg := defaultConfig()
	WithThreadID("thread-abc")(&cfg)

	if cfg.threadID != "thread-abc" {
		t.Errorf("expected 'thread-abc', got %q", cfg.threadID)
	}
}

func TestWithResume(t *testing.T) {
	cfg := defaultConfig()
	WithResume("thread-xyz")(&cfg)

	if cfg.threadID != "thread-xyz" {
		t.Errorf("expected threadID 'thread-xyz', got %q", cfg.threadID)
	}
	if !cfg.resume {
		t.Error("expected resume to be true")
	}
}

func TestWithSystemPrompt(t *testing.T) {
	cfg := defaultConfig()
	WithSystemPrompt("You are helpful.")(&cfg)

	if cfg.systemPrompt != "You are helpful." {
		t.Errorf("expected 'You are helpful.', got %q", cfg.systemPrompt)
	}
}

func TestWithReasoningEffort(t *testing.T) {
	cfg := defaultConfig()
	WithReasoningEffort("high")(&cfg)

	if cfg.reasoningEffort != "high" {
		t.Errorf("expected 'high', got %q", cfg.reasoningEffort)
	}
}

func TestWithEnabledFeatures(t *testing.T) {
	cfg := defaultConfig()
	WithEnabledFeatures([]string{"codex_hooks", "project_doc"})(&cfg)

	if len(cfg.enabledFeatures) != 2 {
		t.Fatalf("expected 2 features, got %d", len(cfg.enabledFeatures))
	}
	if cfg.enabledFeatures[0] != "codex_hooks" {
		t.Errorf("expected 'codex_hooks', got %q", cfg.enabledFeatures[0])
	}
	if cfg.enabledFeatures[1] != "project_doc" {
		t.Errorf("expected 'project_doc', got %q", cfg.enabledFeatures[1])
	}
}

func TestWithDisabledFeatures(t *testing.T) {
	cfg := defaultConfig()
	WithDisabledFeatures([]string{"legacy_mode"})(&cfg)

	if len(cfg.disabledFeatures) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(cfg.disabledFeatures))
	}
	if cfg.disabledFeatures[0] != "legacy_mode" {
		t.Errorf("expected 'legacy_mode', got %q", cfg.disabledFeatures[0])
	}
}

func TestWithStartupTimeout(t *testing.T) {
	cfg := defaultConfig()
	WithStartupTimeout(1 * time.Minute)(&cfg)

	if cfg.startupTimeout != 1*time.Minute {
		t.Errorf("expected 1m, got %v", cfg.startupTimeout)
	}
}

func TestWithIdleTimeout(t *testing.T) {
	cfg := defaultConfig()
	WithIdleTimeout(30 * time.Minute)(&cfg)

	if cfg.idleTimeout != 30*time.Minute {
		t.Errorf("expected 30m, got %v", cfg.idleTimeout)
	}
}

func TestWithEnv(t *testing.T) {
	cfg := defaultConfig()

	WithEnv(map[string]string{"KEY1": "val1", "KEY2": "val2"})(&cfg)
	if cfg.extraEnv["KEY1"] != "val1" {
		t.Errorf("expected KEY1='val1', got %q", cfg.extraEnv["KEY1"])
	}
	if cfg.extraEnv["KEY2"] != "val2" {
		t.Errorf("expected KEY2='val2', got %q", cfg.extraEnv["KEY2"])
	}

	// Calling WithEnv again should merge, not replace.
	WithEnv(map[string]string{"KEY3": "val3"})(&cfg)
	if cfg.extraEnv["KEY1"] != "val1" {
		t.Error("expected KEY1 to be preserved after second WithEnv")
	}
	if cfg.extraEnv["KEY3"] != "val3" {
		t.Errorf("expected KEY3='val3', got %q", cfg.extraEnv["KEY3"])
	}
}

func TestWithEnv_Overwrite(t *testing.T) {
	cfg := defaultConfig()
	WithEnv(map[string]string{"KEY": "original"})(&cfg)
	WithEnv(map[string]string{"KEY": "updated"})(&cfg)

	if cfg.extraEnv["KEY"] != "updated" {
		t.Errorf("expected 'updated', got %q", cfg.extraEnv["KEY"])
	}
}

// =============================================================================
// Manager Options
// =============================================================================

func TestDefaultManagerConfig(t *testing.T) {
	cfg := defaultManagerConfig()

	if cfg.maxSessions != 100 {
		t.Errorf("expected maxSessions 100, got %d", cfg.maxSessions)
	}
	if cfg.sessionTTL != 30*time.Minute {
		t.Errorf("expected sessionTTL 30m, got %v", cfg.sessionTTL)
	}
	if cfg.cleanupInterval != 5*time.Minute {
		t.Errorf("expected cleanupInterval 5m, got %v", cfg.cleanupInterval)
	}
	if len(cfg.defaultOpts) != 0 {
		t.Errorf("expected empty defaultOpts, got %v", cfg.defaultOpts)
	}
}

func TestWithMaxSessions(t *testing.T) {
	cfg := defaultManagerConfig()
	WithMaxSessions(10)(&cfg)

	if cfg.maxSessions != 10 {
		t.Errorf("expected 10, got %d", cfg.maxSessions)
	}
}

func TestWithDefaultSessionOptions(t *testing.T) {
	cfg := defaultManagerConfig()
	opts := []SessionOption{WithModel("test")}
	WithDefaultSessionOptions(opts...)(&cfg)

	if len(cfg.defaultOpts) != 1 {
		t.Errorf("expected 1 default option, got %d", len(cfg.defaultOpts))
	}
}

func TestWithSessionTTL(t *testing.T) {
	cfg := defaultManagerConfig()
	WithSessionTTL(1 * time.Hour)(&cfg)

	if cfg.sessionTTL != 1*time.Hour {
		t.Errorf("expected 1h, got %v", cfg.sessionTTL)
	}
}

func TestWithCleanupInterval(t *testing.T) {
	cfg := defaultManagerConfig()
	WithCleanupInterval(10 * time.Minute)(&cfg)

	if cfg.cleanupInterval != 10*time.Minute {
		t.Errorf("expected 10m, got %v", cfg.cleanupInterval)
	}
}

// =============================================================================
// Multiple Options Compose Correctly
// =============================================================================

func TestOptionsCompose(t *testing.T) {
	cfg := defaultConfig()

	opts := []SessionOption{
		WithCodexPath("/usr/bin/codex"),
		WithModel("o4-mini"),
		WithWorkdir("/project"),
		WithFullAuto(),
		WithReasoningEffort("medium"),
		WithStartupTimeout(15 * time.Second),
		WithIdleTimeout(5 * time.Minute),
		WithEnabledFeatures([]string{"hooks"}),
		WithEnv(map[string]string{"API_KEY": "test"}),
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.codexPath != "/usr/bin/codex" {
		t.Errorf("codexPath: expected '/usr/bin/codex', got %q", cfg.codexPath)
	}
	if cfg.model != "o4-mini" {
		t.Errorf("model: expected 'o4-mini', got %q", cfg.model)
	}
	if cfg.workdir != "/project" {
		t.Errorf("workdir: expected '/project', got %q", cfg.workdir)
	}
	if !cfg.fullAuto {
		t.Error("fullAuto: expected true")
	}
	if cfg.reasoningEffort != "medium" {
		t.Errorf("reasoningEffort: expected 'medium', got %q", cfg.reasoningEffort)
	}
	if cfg.startupTimeout != 15*time.Second {
		t.Errorf("startupTimeout: expected 15s, got %v", cfg.startupTimeout)
	}
	if cfg.idleTimeout != 5*time.Minute {
		t.Errorf("idleTimeout: expected 5m, got %v", cfg.idleTimeout)
	}
	if len(cfg.enabledFeatures) != 1 || cfg.enabledFeatures[0] != "hooks" {
		t.Errorf("enabledFeatures: expected [hooks], got %v", cfg.enabledFeatures)
	}
	if cfg.extraEnv["API_KEY"] != "test" {
		t.Errorf("extraEnv: expected API_KEY='test', got %q", cfg.extraEnv["API_KEY"])
	}
}
