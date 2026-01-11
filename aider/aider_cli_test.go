package aider

import (
	"context"
	"testing"
	"time"

	"github.com/randalmurphal/llmkit/provider"
)

func TestNewAiderCLI(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		wantPath string
	}{
		{
			name:     "default values",
			opts:     nil,
			wantPath: "aider",
		},
		{
			name: "with custom path",
			opts: []Option{
				WithPath("/custom/aider"),
			},
			wantPath: "/custom/aider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cli := NewAiderCLI(tt.opts...)
			if cli.path != tt.wantPath {
				t.Errorf("path = %q, want %q", cli.path, tt.wantPath)
			}
		})
	}
}

func TestAiderCLI_buildArgs(t *testing.T) {
	tests := []struct {
		name   string
		cli    *AiderCLI
		prompt string
		want   []string
	}{
		{
			name:   "minimal args",
			cli:    NewAiderCLI(),
			prompt: "hello world",
			want: []string{
				"--message", "hello world",
				"--no-pretty",
				"--no-fancy-input",
				"--timeout", "300", // 5 minutes default
			},
		},
		{
			name: "with model",
			cli: NewAiderCLI(
				WithModel("ollama_chat/llama3.2"),
			),
			prompt: "test",
			want: []string{
				"--message", "test",
				"--no-pretty",
				"--no-fancy-input",
				"--model", "ollama_chat/llama3.2",
				"--timeout", "300", // 5 minutes default
			},
		},
		{
			name: "with yes always",
			cli: NewAiderCLI(
				WithYesAlways(),
			),
			prompt: "test",
			want: []string{
				"--message", "test",
				"--no-pretty",
				"--no-fancy-input",
				"--timeout", "300",
				"--yes-always",
			},
		},
		{
			name: "with git options",
			cli: NewAiderCLI(
				WithNoGit(),
				WithNoAutoCommits(),
			),
			prompt: "test",
			want: []string{
				"--message", "test",
				"--no-pretty",
				"--no-fancy-input",
				"--timeout", "300",
				"--no-git",
				"--no-auto-commits",
			},
		},
		{
			name: "with files",
			cli: NewAiderCLI(
				WithEditableFiles([]string{"main.go", "utils.go"}),
				WithReadOnlyFiles([]string{"README.md"}),
			),
			prompt: "test",
			want: []string{
				"--message", "test",
				"--no-pretty",
				"--no-fancy-input",
				"--timeout", "300",
				"--file", "main.go",
				"--file", "utils.go",
				"--read", "README.md",
			},
		},
		{
			name: "with dry run",
			cli: NewAiderCLI(
				WithDryRun(),
			),
			prompt: "test",
			want: []string{
				"--message", "test",
				"--no-pretty",
				"--no-fancy-input",
				"--timeout", "300",
				"--dry-run",
			},
		},
		{
			name: "with edit format",
			cli: NewAiderCLI(
				WithEditFormat("diff"),
			),
			prompt: "test",
			want: []string{
				"--message", "test",
				"--no-pretty",
				"--no-fancy-input",
				"--timeout", "300",
				"--edit-format", "diff",
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

func TestAiderCLI_buildEnv(t *testing.T) {
	cli := NewAiderCLI(
		WithOllamaAPIBase("http://localhost:11434"),
		WithEnv(map[string]string{"CUSTOM_VAR": "value"}),
	)

	env := cli.buildEnv()

	// Check for Ollama API base
	foundOllama := false
	foundCustomVar := false
	for _, e := range env {
		if e == "OLLAMA_API_BASE=http://localhost:11434" {
			foundOllama = true
		}
		if e == "CUSTOM_VAR=value" {
			foundCustomVar = true
		}
	}

	if !foundOllama {
		t.Error("Ollama API base not found in environment")
	}
	if !foundCustomVar {
		t.Error("Custom var not found in environment")
	}
}

func TestAiderCLI_extractPrompt(t *testing.T) {
	cli := NewAiderCLI()

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

func TestAiderCLI_Capabilities(t *testing.T) {
	cli := NewAiderCLI()
	caps := cli.Capabilities()

	if !caps.Streaming {
		t.Error("expected Streaming = true")
	}
	if !caps.Tools {
		t.Error("expected Tools = true")
	}
	if caps.MCP {
		t.Error("expected MCP = false (not yet supported)")
	}
	if caps.Sessions {
		t.Error("expected Sessions = false")
	}
	if caps.Images {
		t.Error("expected Images = false")
	}
	if len(caps.NativeTools) == 0 {
		t.Error("expected NativeTools to be populated")
	}
}

func TestAiderCLI_Provider(t *testing.T) {
	cli := NewAiderCLI()
	if cli.Provider() != "aider" {
		t.Errorf("Provider() = %q, want %q", cli.Provider(), "aider")
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
			cfg:     Config{Path: "aider", Timeout: -1 * time.Second},
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

	if cfg.Path != "aider" {
		t.Errorf("Path = %q, want %q", cfg.Path, "aider")
	}
	if cfg.Timeout != 5*time.Minute {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, 5*time.Minute)
	}
}

func TestParseAiderOutput(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "simple output",
			output: "I've made the changes you requested.\n",
			want:   "I've made the changes you requested.",
		},
		{
			name:   "with whitespace",
			output: "  \n  Hello world  \n  ",
			want:   "Hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ParseAiderOutput(tt.output)
			if resp.Content != tt.want {
				t.Errorf("Content = %q, want %q", resp.Content, tt.want)
			}
		})
	}
}

func TestParseEditMarkers(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []EditMarker
	}{
		{
			name:   "applied edit",
			output: "Applied edit to main.go",
			want:   []EditMarker{{Action: "applied", File: "main.go"}},
		},
		{
			name:   "created file",
			output: "Created new_file.go",
			want:   []EditMarker{{Action: "created", File: "new_file.go"}},
		},
		{
			name:   "wrote file",
			output: "Wrote utils.go",
			want:   []EditMarker{{Action: "modified", File: "utils.go"}},
		},
		{
			name:   "multiple edits",
			output: "Applied edit to main.go\nCreated test.go\nWrote utils.go",
			want: []EditMarker{
				{Action: "applied", File: "main.go"},
				{Action: "created", File: "test.go"},
				{Action: "modified", File: "utils.go"},
			},
		},
		{
			name:   "no edits",
			output: "I understand your request.",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseEditMarkers(tt.output)
			if len(got) != len(tt.want) {
				t.Errorf("markers length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i, marker := range got {
				if marker.Action != tt.want[i].Action || marker.File != tt.want[i].File {
					t.Errorf("marker[%d] = %+v, want %+v", i, marker, tt.want[i])
				}
			}
		})
	}
}

func TestContainsCommit(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   bool
	}{
		{
			name:   "has commit",
			output: "Committed abc1234",
			want:   true,
		},
		{
			name:   "no commit",
			output: "Made changes to file",
			want:   false,
		},
		{
			name:   "commit hash pattern",
			output: "commit a1b2c3d4e5f6",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContainsCommit(tt.output)
			if got != tt.want {
				t.Errorf("ContainsCommit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractCommitHash(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "short hash",
			output: "Committed abc1234",
			want:   "abc1234",
		},
		{
			name:   "full hash",
			output: "commit a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
			want:   "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
		},
		{
			name:   "no hash",
			output: "No commit made",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCommitHash(tt.output)
			if got != tt.want {
				t.Errorf("ExtractCommitHash() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestAiderCLI_Complete_NotInstalled tests behavior when aider is not installed.
func TestAiderCLI_Complete_NotInstalled(t *testing.T) {
	cli := NewAiderCLI(
		WithPath("/nonexistent/aider"),
	)

	_, err := cli.Complete(context.Background(), provider.Request{
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "test"},
		},
	})

	if err == nil {
		t.Error("expected error when aider not installed")
	}
}
