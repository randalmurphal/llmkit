// Package claudecontract provides a single source of truth for all Claude CLI
// interface details including flag names, event types, file paths, and other
// volatile strings that may change between CLI versions.
//
// Both the claude/ and claudeconfig/ packages import from here to ensure
// consistency and make updates easier when the CLI changes.
package claudecontract

import (
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// TestedCLIVersion is the Claude CLI version this code was tested against.
// When the detected CLI version is newer, a warning is logged.
const TestedCLIVersion = "2.1.19"

// CLIVersion represents a parsed Claude CLI version.
type CLIVersion struct {
	Major int
	Minor int
	Patch int
	Raw   string
}

// ParseVersion parses a version string like "2.1.19" into a CLIVersion.
func ParseVersion(s string) (*CLIVersion, error) {
	// Handle formats like "2.1.19 (Claude Code)" or just "2.1.19"
	s = strings.TrimSpace(s)
	parts := strings.Fields(s)
	if len(parts) > 0 {
		s = parts[0]
	}

	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return nil, fmt.Errorf("invalid version format: %q", s)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return &CLIVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
		Raw:   s,
	}, nil
}

// MustParseVersion parses a version string, panicking on error.
// Use only for known-good version constants.
func MustParseVersion(s string) *CLIVersion {
	v, err := ParseVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

// DetectCLIVersion runs the claude binary and parses its version.
func DetectCLIVersion(claudePath string) (*CLIVersion, error) {
	if claudePath == "" {
		claudePath = "claude"
	}

	out, err := exec.Command(claudePath, "--version").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run %s --version: %w", claudePath, err)
	}

	return ParseVersion(string(out))
}

// String returns the version as a string.
func (v *CLIVersion) String() string {
	return v.Raw
}

// Compare returns -1 if v < other, 0 if v == other, 1 if v > other.
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

// IsNewerThan returns true if v is newer than other.
func (v *CLIVersion) IsNewerThan(other *CLIVersion) bool {
	return v.Compare(other) > 0
}

// IsOlderThan returns true if v is older than other.
func (v *CLIVersion) IsOlderThan(other *CLIVersion) bool {
	return v.Compare(other) < 0
}

// WarnIfUntested logs a warning if this version is newer than the tested version.
func (v *CLIVersion) WarnIfUntested() {
	tested := MustParseVersion(TestedCLIVersion)
	if v.IsNewerThan(tested) {
		slog.Warn("Claude CLI version is newer than SDK tested version",
			"cli_version", v.Raw,
			"tested_version", TestedCLIVersion,
			"note", "Some features may not work as expected",
		)
	}
}

// CheckVersion detects the CLI version and warns if untested.
// Returns the detected version or nil if detection fails.
func CheckVersion(claudePath string) *CLIVersion {
	v, err := DetectCLIVersion(claudePath)
	if err != nil {
		slog.Debug("Could not detect Claude CLI version", "error", err)
		return nil
	}
	v.WarnIfUntested()
	return v
}
