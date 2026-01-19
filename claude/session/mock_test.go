package session

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// TestMain sets up the test helper command for mocking Claude CLI.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

// mockSession implements Session for testing without real CLI.
type mockSession struct {
	id           string
	status       SessionStatus
	outputCh     chan OutputMessage
	closed       bool
	closeMu      sync.Mutex
	sendMessages []UserMessage
	info         SessionInfo
}

func newMockSession(id string) *mockSession {
	return &mockSession{
		id:       id,
		status:   StatusActive,
		outputCh: make(chan OutputMessage, 100),
		info: SessionInfo{
			ID:           id,
			Status:       StatusActive,
			Model:        "claude-opus-4-5-20251101",
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		},
	}
}

func (m *mockSession) ID() string                          { return m.id }
func (m *mockSession) Output() <-chan OutputMessage        { return m.outputCh }
func (m *mockSession) Status() SessionStatus               { return m.status }
func (m *mockSession) Info() SessionInfo                   { return m.info }
func (m *mockSession) Wait() error                         { return nil }
func (m *mockSession) WaitForInit(_ context.Context) error { return nil }
func (m *mockSession) JSONLPath() string                   { return "" }

func (m *mockSession) Send(_ context.Context, msg UserMessage) error {
	m.closeMu.Lock()
	defer m.closeMu.Unlock()
	if m.closed {
		return fmt.Errorf("session closed")
	}
	m.sendMessages = append(m.sendMessages, msg)
	return nil
}

func (m *mockSession) Close() error {
	m.closeMu.Lock()
	defer m.closeMu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	m.status = StatusClosed
	close(m.outputCh)
	return nil
}

// simulateOutput sends a mock output message.
func (m *mockSession) simulateOutput(msg OutputMessage) {
	m.outputCh <- msg
}

// mockSessionManager implements SessionManager for testing.
type mockSessionManager struct {
	sessions     map[string]*mockSession
	mu           sync.RWMutex
	createError  error
	maxSessions  int
	sessionCount int
}

func newMockSessionManager() *mockSessionManager {
	return &mockSessionManager{
		sessions:    make(map[string]*mockSession),
		maxSessions: 100,
	}
}

func (m *mockSessionManager) Create(_ context.Context, _ ...SessionOption) (Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.createError != nil {
		return nil, m.createError
	}
	if len(m.sessions) >= m.maxSessions {
		return nil, fmt.Errorf("max sessions reached")
	}

	m.sessionCount++
	id := fmt.Sprintf("mock-session-%d", m.sessionCount)
	sess := newMockSession(id)
	m.sessions[id] = sess
	return sess, nil
}

func (m *mockSessionManager) Get(sessionID string) (Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok || s.status != StatusActive {
		return nil, false
	}
	return s, true
}

func (m *mockSessionManager) Resume(ctx context.Context, sessionID string, opts ...SessionOption) (Session, error) {
	if s, ok := m.Get(sessionID); ok {
		return s, nil
	}
	return m.Create(ctx, opts...)
}

func (m *mockSessionManager) Close(sessionID string) error {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("session not found")
	}
	return s.Close()
}

func (m *mockSessionManager) CloseAll() error {
	m.mu.Lock()
	sessions := make([]*mockSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.mu.Unlock()

	for _, s := range sessions {
		_ = s.Close()
	}
	return nil
}

func (m *mockSessionManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}

func (m *mockSessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

func (m *mockSessionManager) Info(sessionID string) (*SessionInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}
	info := s.Info()
	return &info, true
}
