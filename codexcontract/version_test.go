package codexcontract

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0.98.0", "0.98.0"},
		{"codex-cli 0.98.0", "0.98.0"},
		{"Codex CLI version 1.2.3 (build abc)", "1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v, err := ParseVersion(tt.input)
			if err != nil {
				t.Fatalf("ParseVersion returned error: %v", err)
			}
			if v.Raw != tt.want {
				t.Fatalf("ParseVersion(%q) = %q, want %q", tt.input, v.Raw, tt.want)
			}
		})
	}
}

func TestDetectCLIVersion(t *testing.T) {
	codexPath, err := findCodexCLI()
	if err != nil {
		t.Skip("codex CLI not found")
	}

	v, err := DetectCLIVersion(codexPath)
	if err != nil {
		t.Fatalf("DetectCLIVersion returned error: %v", err)
	}
	if v.Raw == "" {
		t.Fatal("detected version is empty")
	}
}
