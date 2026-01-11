package continuedev

import (
	"context"
	"testing"
	"time"

	"github.com/randalmurphal/llmkit/provider"
)

func TestNewContinueCLI(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		wantPath string
	}{
		{
			name:     "default values",
			opts:     nil,
			wantPath: "cn",
		},
		{
			name: "with custom path",
			opts: []Option{
				WithPath("/custom/cn"),
			},
			wantPath: "/custom/cn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewContinueCLI(tt.opts...)
			if cli.path != tt.wantPath {
				t.Errorf("path = %q, want %q", cli.path, tt.wantPath)
			}
		})
	}
}

func TestContinueCLI_buildArgs(t *testing.T) {
	tests := []struct {
		name   string
		cli    *ContinueCLI
		prompt string
		want   []string
	}{
		{
			name:   "minimal args",
			cli:    NewContinueCLI(),
			prompt: "hello world",
			want:   []string{"-p", "hello world"},
		},
		{
			name: "with config path",
			cli: NewContinueCLI(
				WithConfigPath("~/.continue/config.yaml"),
			),
			prompt: "test",
			want:   []string{"-p", "test", "--config", "~/.continue/config.yaml"},
		},
		{
			name: "with resume",
			cli: NewContinueCLI(
				WithResume(),
			),
			prompt: "test",
			want:   []string{"-p", "test", "--resume"},
		},
		{
			name: "with rule",
			cli: NewContinueCLI(
				WithRule("nate/spanish"),
			),
			prompt: "test",
			want:   []string{"-p", "test", "--rule", "nate/spanish"},
		},
		{
			name: "with verbose",
			cli: NewContinueCLI(
				WithVerbose(),
			),
			prompt: "test",
			want:   []string{"-p", "test", "--verbose"},
		},
		{
			name: "with tool permissions",
			cli: NewContinueCLI(
				WithAllowedTools([]string{"Write()"}),
				WithAskTools([]string{"Bash(*)"}),
				WithExcludedTools([]string{"Fetch"}),
			),
			prompt: "test",
			want: []string{
				"-p", "test",
				"--allow", "Write()",
				"--ask", "Bash(*)",
				"--exclude", "Fetch",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cli.buildArgs(tt.prompt)
			if len(got) != len(tt.want) {
				t.Errorf("args length = %d, want %d\ngot: %v\nwant: %v", len(got), len(tt.want), got, tt.want)
				return
			}
			for i, arg := range got {
				if arg != tt.want[i] {
					t.Errorf("args[%d] = %q, want %q", i, arg, tt.want[i])
				}
			}
		})
	}
}

func TestContinueCLI_buildEnv(t *testing.T) {
	cli := NewContinueCLI(
		WithAPIKey("test-api-key"),
		WithEnv(map[string]string{"CUSTOM_VAR": "value"}),
	)

	env := cli.buildEnv()

	// Check for API key
	foundAPIKey := false
	foundCustomVar := false
	for _, e := range env {
		if e == "CONTINUE_API_KEY=test-api-key" {
			foundAPIKey = true
		}
		if e == "CUSTOM_VAR=value" {
			foundCustomVar = true
		}
	}

	if !foundAPIKey {
		t.Error("API key not found in environment")
	}
	if !foundCustomVar {
		t.Error("Custom var not found in environment")
	}
}

func TestContinueCLI_parseResponse(t *testing.T) {
	tests := []struct {
		name    string
		output  []byte
		want    string
		wantErr bool
	}{
		{
			name:   "plain text response",
			output: []byte("Hello, I can help with that.\n"),
			want:   "Hello, I can help with that.",
		},
		{
			name:   "json response",
			output: []byte(`{"content": "JSON content", "finish_reason": "stop", "usage": {"input_tokens": 10, "output_tokens": 5}}`),
			want:   "JSON content",
		},
		{
			name:    "empty response",
			output:  []byte(""),
			wantErr: true,
		},
		{
			name:    "whitespace only",
			output:  []byte("   \n  \t  "),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewContinueCLI()
			resp, err := cli.parseResponse(tt.output)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if resp.Content != tt.want {
				t.Errorf("content = %q, want %q", resp.Content, tt.want)
			}
		})
	}
}

func TestContinueCLI_extractPrompt(t *testing.T) {
	cli := NewContinueCLI()

	tests := []struct {
		name string
		req  provider.Request
		want string
	}{
		{
			name: "user message only",
			req: provider.Request{
				Messages: []provider.Message{
					{Role: provider.RoleUser, Content: "Hello"},
				},
			},
			want: "Hello",
		},
		{
			name: "with system prompt",
			req: provider.Request{
				SystemPrompt: "You are helpful",
				Messages: []provider.Message{
					{Role: provider.RoleUser, Content: "Hello"},
				},
			},
			want: "You are helpful\n\nHello",
		},
		{
			name: "multiple messages - uses last user",
			req: provider.Request{
				Messages: []provider.Message{
					{Role: provider.RoleUser, Content: "First"},
					{Role: provider.RoleAssistant, Content: "Response"},
					{Role: provider.RoleUser, Content: "Second"},
				},
			},
			want: "Second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cli.extractPrompt(tt.req)
			if got != tt.want {
				t.Errorf("prompt = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContinueCLI_Capabilities(t *testing.T) {
	cli := NewContinueCLI()
	caps := cli.Capabilities()

	if !caps.Streaming {
		t.Error("expected Streaming = true")
	}
	if !caps.Tools {
		t.Error("expected Tools = true")
	}
	if !caps.MCP {
		t.Error("expected MCP = true")
	}
	if !caps.Sessions {
		t.Error("expected Sessions = true")
	}
	if !caps.Images {
		t.Error("expected Images = true")
	}
	if len(caps.NativeTools) == 0 {
		t.Error("expected NativeTools to be populated")
	}
}

func TestContinueCLI_Provider(t *testing.T) {
	cli := NewContinueCLI()
	if cli.Provider() != "continue" {
		t.Errorf("Provider() = %q, want %q", cli.Provider(), "continue")
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
			name:    "empty path",
			cfg:     Config{Path: ""},
			wantErr: true,
		},
		{
			name:    "negative timeout",
			cfg:     Config{Path: "cn", Timeout: -1 * time.Second},
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

func TestConfig_WithDefaults(t *testing.T) {
	cfg := Config{}
	cfg = cfg.WithDefaults()

	if cfg.Path != "cn" {
		t.Errorf("Path = %q, want %q", cfg.Path, "cn")
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 5*time.Minute)
	}
}

// TestContinueCLI_Complete_NotInstalled tests behavior when cn is not installed.
func TestContinueCLI_Complete_NotInstalled(t *testing.T) {
	cli := NewContinueCLI(
		WithPath("/nonexistent/cn"),
	)

	_, err := cli.Complete(context.Background(), provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "test"},
		},
	})

	if err == nil {
		t.Error("expected error when cn not installed")
	}
}
