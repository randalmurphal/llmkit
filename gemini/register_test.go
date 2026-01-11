package gemini_test

import (
	"testing"

	// Import gemini package to trigger init() registration
	_ "github.com/randalmurphal/llmkit/gemini"
	"github.com/randalmurphal/llmkit/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiProviderRegistration(t *testing.T) {
	// Verify gemini provider is registered
	assert.True(t, provider.IsRegistered("gemini"), "gemini provider should be registered")
}

func TestGeminiProviderAvailable(t *testing.T) {
	// Verify gemini appears in available providers
	available := provider.Available()
	found := false
	for _, name := range available {
		if name == "gemini" {
			found = true
			break
		}
	}
	assert.True(t, found, "gemini should be in available providers list")
}

func TestGeminiProviderNew(t *testing.T) {
	// Create gemini client via provider registry
	cfg := provider.Config{
		Provider: "gemini",
		Model:    "gemini-2.5-pro",
	}

	client, err := provider.New("gemini", cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	// Verify provider name
	assert.Equal(t, "gemini", client.Provider())

	// Verify capabilities
	caps := client.Capabilities()
	assert.True(t, caps.Streaming)
	assert.True(t, caps.Tools)
	assert.True(t, caps.MCP)
	assert.False(t, caps.Sessions)
	assert.True(t, caps.Images)
	assert.Equal(t, "GEMINI.md", caps.ContextFile)

	// Verify native tools
	assert.True(t, caps.HasTool("read_file"))
	assert.True(t, caps.HasTool("write_file"))
	assert.True(t, caps.HasTool("run_shell_command"))
	assert.True(t, caps.HasTool("web_fetch"))
	assert.True(t, caps.HasTool("google_web_search"))
}

func TestGeminiProviderWithOptions(t *testing.T) {
	// Test provider creation with Gemini-specific options
	cfg := provider.Config{
		Provider:        "gemini",
		Model:           "gemini-2.5-pro",
		SystemPrompt:    "You are a helpful assistant",
		MaxTurns:        10,
		WorkDir:         "/tmp",
		AllowedTools:    []string{"read_file"},
		DisallowedTools: []string{"write_file"},
		Options: map[string]any{
			"yolo":      true,
			"sandbox":   "docker",
			"gemini_path": "/custom/path/gemini",
		},
	}

	client, err := provider.New("gemini", cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	assert.Equal(t, "gemini", client.Provider())
}

func TestGeminiProviderValidation(t *testing.T) {
	// Test that provider config validation works
	cfg := provider.Config{
		Provider: "", // Invalid - missing provider
		Model:    "gemini-2.5-pro",
	}

	_, err := provider.New("gemini", cfg)
	assert.Error(t, err, "should fail with missing provider in config")
}
