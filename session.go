package llmkit

import (
	"encoding/json"
	"fmt"
)

func SessionMetadataForID(provider, id string) *SessionMetadata {
	if provider == "" || id == "" {
		return nil
	}
	data, _ := json.Marshal(map[string]string{"session_id": id})
	return &SessionMetadata{
		Provider: provider,
		Data:     data,
	}
}

func SessionID(session *SessionMetadata) string {
	if session == nil {
		return ""
	}
	var payload struct {
		ID        string `json:"id"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(session.Data, &payload); err != nil {
		return ""
	}
	if payload.SessionID != "" {
		return payload.SessionID
	}
	return payload.ID
}

func MarshalSessionMetadata(session *SessionMetadata) (string, error) {
	if session == nil {
		return "", nil
	}
	data, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("marshal session metadata: %w", err)
	}
	return string(data), nil
}

func ParseSessionMetadata(raw string) (*SessionMetadata, error) {
	if raw == "" {
		return nil, nil
	}
	var session SessionMetadata
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return nil, fmt.Errorf("parse session metadata: %w", err)
	}
	return &session, nil
}
