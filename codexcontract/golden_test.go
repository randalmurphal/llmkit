package codexcontract

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGoldenHelpOutput(t *testing.T) {
	codexPath, err := findCodexCLI()
	if err != nil {
		t.Skip("codex CLI not found")
	}

	version, err := DetectCLIVersion(codexPath)
	if err != nil {
		t.Fatalf("failed to detect codex version: %v", err)
	}

	tests := []struct {
		name string
		args []string
		file string
	}{
		{name: "root-help", args: []string{"--help"}, file: "help_v" + version.Raw + ".txt"},
		{name: "exec-help", args: []string{"exec", "--help"}, file: "exec_help_v" + version.Raw + ".txt"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			currentOutput := runCodexOutput(t, codexPath, tc.args...)
			goldenPath := filepath.Join("testdata", tc.file)

			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatalf("failed to create testdata dir: %v", err)
				}
				if err := os.WriteFile(goldenPath, []byte(currentOutput), 0o644); err != nil {
					t.Fatalf("failed to write golden file: %v", err)
				}
				t.Logf("updated golden file: %s", goldenPath)
				return
			}

			goldenOutput, err := os.ReadFile(goldenPath)
			if err != nil {
				if os.IsNotExist(err) {
					t.Logf("golden file not found: %s", goldenPath)
					t.Log("run with UPDATE_GOLDEN=1 to create it:")
					t.Log("  UPDATE_GOLDEN=1 go test ./codexcontract -run TestGoldenHelpOutput")
					t.Skip("golden file does not exist")
				}
				t.Fatalf("failed to read golden file: %v", err)
			}

			if !bytes.Equal([]byte(currentOutput), goldenOutput) {
				t.Errorf("help output differs from golden file %s", goldenPath)
				t.Log("run with UPDATE_GOLDEN=1 to update if expected")
			}
		})
	}
}
