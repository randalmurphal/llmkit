package llmkit

import "testing"

func TestSessionMetadataRoundTrip(t *testing.T) {
	session := SessionMetadataForID("codex", "thread-123")
	if session == nil {
		t.Fatal("expected session metadata")
	}

	if got := SessionID(session); got != "thread-123" {
		t.Fatalf("SessionID() = %q, want %q", got, "thread-123")
	}

	raw, err := MarshalSessionMetadata(session)
	if err != nil {
		t.Fatalf("MarshalSessionMetadata() error = %v", err)
	}

	parsed, err := ParseSessionMetadata(raw)
	if err != nil {
		t.Fatalf("ParseSessionMetadata() error = %v", err)
	}
	if parsed.Provider != "codex" {
		t.Fatalf("provider = %q, want codex", parsed.Provider)
	}
	if got := SessionID(parsed); got != "thread-123" {
		t.Fatalf("round-trip SessionID() = %q, want %q", got, "thread-123")
	}
}
