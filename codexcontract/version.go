package codexcontract

import (
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// TestedCLIVersion is the Codex CLI version this package was validated against.
const TestedCLIVersion = "0.98.0"

// CLIVersion represents a parsed semver-like CLI version.
type CLIVersion struct {
	Major int
	Minor int
	Patch int
	Raw   string
}

// ParseVersion parses versions like "0.34.1".
func ParseVersion(s string) (*CLIVersion, error) {
	s = strings.TrimSpace(s)

	re := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		return nil, fmt.Errorf("invalid version format: %q", s)
	}

	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])

	return &CLIVersion{Major: major, Minor: minor, Patch: patch, Raw: m[0]}, nil
}

// MustParseVersion parses a version and panics on invalid input.
func MustParseVersion(s string) *CLIVersion {
	v, err := ParseVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

// DetectCLIVersion runs codex --version and parses the result.
func DetectCLIVersion(codexPath string) (*CLIVersion, error) {
	if codexPath == "" {
		codexPath = "codex"
	}
	out, err := exec.Command(codexPath, FlagVersion).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to run %s --version: %w", codexPath, err)
	}
	return ParseVersion(string(out))
}

// Compare returns -1 when v < other, 0 when equal, and 1 when v > other.
func (v *CLIVersion) Compare(other *CLIVersion) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// IsNewerThan reports whether v is newer than other.
func (v *CLIVersion) IsNewerThan(other *CLIVersion) bool {
	return v.Compare(other) > 0
}

// WarnIfUntested warns when runtime CLI is newer than tested version.
func (v *CLIVersion) WarnIfUntested() {
	tested := MustParseVersion(TestedCLIVersion)
	if v.IsNewerThan(tested) {
		slog.Warn("Codex CLI version is newer than SDK tested version",
			"cli_version", v.Raw,
			"tested_version", TestedCLIVersion,
			"note", "headless event schema may have changed")
	}
}

// CheckVersion detects CLI version and logs compatibility warnings.
func CheckVersion(codexPath string) *CLIVersion {
	v, err := DetectCLIVersion(codexPath)
	if err != nil {
		slog.Debug("could not detect Codex CLI version", "error", err)
		return nil
	}
	v.WarnIfUntested()
	return v
}
