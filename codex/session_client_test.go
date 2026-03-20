package codex_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/randalmurphal/llmkit/codex"
	"github.com/randalmurphal/llmkit/codex/session"
	"github.com/randalmurphal/llmkit/codexcontract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Mock Session and Manager for testing SessionClient
// =============================================================================

type testSession struct {
	id           string
	threadID     string
	status       session.SessionStatus
	outputCh     chan session.OutputMessage
	closed       bool
	closeMu      sync.Mutex
	sendMessages []session.UserMessage
	steerCalls   []session.UserMessage
	info         session.SessionInfo
	sendErr      error
	steerErr     error
}

func newTestSession(id string) *testSession {
	return &testSession{
		id:       id,
		threadID: id,
		status:   session.StatusActive,
		outputCh: make(chan session.OutputMessage, 100),
		info: session.SessionInfo{
			ID:           id,
			ThreadID:     id,
			Status:       session.StatusActive,
			Model:        "o4-mini",
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		},
	}
}

func (s *testSession) ID() string       { return s.id }
func (s *testSession) ThreadID() string  { return s.threadID }
func (s *testSession) Status() session.SessionStatus { return s.status }
func (s *testSession) Info() session.SessionInfo      { return s.info }
func (s *testSession) Wait() error                    { return nil }
func (s *testSession) WaitForInit(_ context.Context) error { return nil }
func (s *testSession) Output() <-chan session.OutputMessage { return s.outputCh }
func (s *testSession) JSONLPath() string { return "" }

func (s *testSession) Send(_ context.Context, msg session.UserMessage) error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.sendErr != nil {
		return s.sendErr
	}
	if s.closed {
		return fmt.Errorf("session closed")
	}
	s.sendMessages = append(s.sendMessages, msg)
	return nil
}

func (s *testSession) Steer(_ context.Context, msg session.UserMessage) error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.steerErr != nil {
		return s.steerErr
	}
	s.steerCalls = append(s.steerCalls, msg)
	return nil
}

func (s *testSession) Close() error {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	s.status = session.StatusClosed
	close(s.outputCh)
	return nil
}

func (s *testSession) simulateOutput(msgs ...session.OutputMessage) {
	for _, msg := range msgs {
		s.outputCh <- msg
	}
}

type testSessionManager struct {
	sessions    map[string]*testSession
	mu          sync.RWMutex
	createError error
	nextID      int
}

func newTestSessionManager() *testSessionManager {
	return &testSessionManager{
		sessions: make(map[string]*testSession),
	}
}

func (m *testSessionManager) Create(_ context.Context, _ ...session.SessionOption) (session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createError != nil {
		return nil, m.createError
	}
	m.nextID++
	id := fmt.Sprintf("test-session-%d", m.nextID)
	sess := newTestSession(id)
	m.sessions[id] = sess
	return sess, nil
}

func (m *testSessionManager) Get(sessionID string) (session.Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}
	return s, true
}

func (m *testSessionManager) Resume(ctx context.Context, threadID string, opts ...session.SessionOption) (session.Session, error) {
	if s, ok := m.Get(threadID); ok {
		return s, nil
	}
	return m.Create(ctx, opts...)
}

func (m *testSessionManager) Close(sessionID string) error {
	m.mu.Lock()
	s, ok := m.sessions[sessionID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("session not found")
	}
	return s.Close()
}

func (m *testSessionManager) CloseAll() error {
	m.mu.Lock()
	sessions := make([]*testSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.mu.Unlock()
	for _, s := range sessions {
		_ = s.Close()
	}
	return nil
}

func (m *testSessionManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}

func (m *testSessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

func (m *testSessionManager) Info(sessionID string) (*session.SessionInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}
	info := s.Info()
	return &info, true
}

// Add a test session to the manager (for pre-populating).
func (m *testSessionManager) addSession(s *testSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.id] = s
}

// =============================================================================
// NewSessionClient Tests
// =============================================================================

func TestNewSessionClient(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	require.NotNil(t, client)
	assert.Equal(t, "sess-1", client.SessionID())
	assert.Equal(t, "sess-1", client.ThreadID())
	assert.NotNil(t, client.Session())
}

func TestNewSessionClient_Close_DoesNotCloseUnowned(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	err := client.Close()
	assert.NoError(t, err)

	// Session should still be open since client doesn't own it.
	assert.Equal(t, session.StatusActive, sess.Status())
}

// =============================================================================
// Complete Tests
// =============================================================================

func TestSessionClient_Complete_CollectsOutput(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	// Simulate output arriving after Send is called.
	go func() {
		// Wait a bit for Send to be called.
		time.Sleep(10 * time.Millisecond)
		sess.simulateOutput(
			session.OutputMessage{
				Type:     codexcontract.EventItemUpdated,
				ItemType: codexcontract.ItemAgentMessage,
				Content:  "Hello ",
			},
			session.OutputMessage{
				Type:     codexcontract.EventItemUpdated,
				ItemType: codexcontract.ItemAgentMessage,
				Content:  "World!",
			},
			session.OutputMessage{
				Type: codexcontract.EventTurnCompleted,
			},
		)
	}()

	resp, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "Hi"}},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Hello World!", resp.Content)
	assert.Equal(t, "sess-1", resp.SessionID)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.True(t, resp.Duration > 0)
}

func TestSessionClient_Complete_EmptyMessages(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	_, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no message content")
}

func TestSessionClient_Complete_SendError(t *testing.T) {
	sess := newTestSession("sess-1")
	sess.sendErr = fmt.Errorf("connection lost")
	client := codex.NewSessionClient(sess)

	_, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "test"}},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send message")
}

func TestSessionClient_Complete_ContextCancellation(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after a short delay.
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := client.Complete(ctx, codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "test"}},
	})

	assert.Error(t, err)
}

func TestSessionClient_Complete_TurnFailed(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	go func() {
		time.Sleep(10 * time.Millisecond)
		sess.simulateOutput(
			session.OutputMessage{
				Type:     codexcontract.EventItemUpdated,
				ItemType: codexcontract.ItemAgentMessage,
				Content:  "partial",
			},
			session.OutputMessage{
				Type:  codexcontract.EventTurnFailed,
				Error: "rate limited",
			},
		)
	}()

	resp, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "test"}},
	})

	// TurnFailed still returns a response (with partial content).
	require.NoError(t, err)
	assert.Equal(t, "partial", resp.Content)
}

func TestSessionClient_Complete_ChannelClosedEarly(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	go func() {
		time.Sleep(10 * time.Millisecond)
		// Close the output channel without sending turn complete.
		close(sess.outputCh)
	}()

	resp, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "test"}},
	})

	// Should still return a response with whatever was collected.
	require.NoError(t, err)
	assert.Equal(t, "", resp.Content)
}

func TestSessionClient_Complete_SetsModel(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	go func() {
		time.Sleep(10 * time.Millisecond)
		sess.simulateOutput(session.OutputMessage{
			Type: codexcontract.EventTurnCompleted,
		})
	}()

	resp, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "test"}},
		Model:    "o4-mini",
	})

	require.NoError(t, err)
	assert.Equal(t, "o4-mini", resp.Model)
}

func TestSessionClient_Complete_IgnoresNonAgentMessages(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	go func() {
		time.Sleep(10 * time.Millisecond)
		sess.simulateOutput(
			// Reasoning item should be ignored.
			session.OutputMessage{
				Type:     codexcontract.EventItemUpdated,
				ItemType: codexcontract.ItemReasoning,
				Content:  "thinking...",
			},
			// Agent message should be captured.
			session.OutputMessage{
				Type:     codexcontract.EventItemUpdated,
				ItemType: codexcontract.ItemAgentMessage,
				Content:  "actual response",
			},
			session.OutputMessage{
				Type: codexcontract.EventTurnCompleted,
			},
		)
	}()

	resp, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "test"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "actual response", resp.Content)
}

// =============================================================================
// Stream Tests
// =============================================================================

func TestSessionClient_Stream_DeliversChunks(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	go func() {
		time.Sleep(10 * time.Millisecond)
		sess.simulateOutput(
			session.OutputMessage{
				Type:     codexcontract.EventItemUpdated,
				ItemType: codexcontract.ItemAgentMessage,
				Content:  "chunk1",
			},
			session.OutputMessage{
				Type:     codexcontract.EventItemUpdated,
				ItemType: codexcontract.ItemAgentMessage,
				Content:  "chunk2",
			},
			session.OutputMessage{
				Type: codexcontract.EventTurnCompleted,
			},
		)
	}()

	ch, err := client.Stream(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "test"}},
	})

	require.NoError(t, err)

	var chunks []codex.StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should have content chunks + done chunk.
	require.True(t, len(chunks) >= 3, "expected at least 3 chunks, got %d", len(chunks))

	assert.Equal(t, "chunk1", chunks[0].Content)
	assert.Equal(t, "sess-1", chunks[0].SessionID)
	assert.Equal(t, "chunk2", chunks[1].Content)

	// Last chunk should be done.
	lastChunk := chunks[len(chunks)-1]
	assert.True(t, lastChunk.Done)
}

func TestSessionClient_Stream_EmptyMessages(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	_, err := client.Stream(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no message content")
}

func TestSessionClient_Stream_TurnFailed(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	go func() {
		time.Sleep(10 * time.Millisecond)
		sess.simulateOutput(
			session.OutputMessage{
				Type:  codexcontract.EventTurnFailed,
				Error: "something broke",
			},
		)
	}()

	ch, err := client.Stream(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "test"}},
	})
	require.NoError(t, err)

	var chunks []codex.StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	require.True(t, len(chunks) >= 1)
	lastChunk := chunks[len(chunks)-1]
	assert.True(t, lastChunk.Done)
	assert.Error(t, lastChunk.Error)
	assert.Contains(t, lastChunk.Error.Error(), "turn failed")
}

func TestSessionClient_Stream_ChannelClosedWithoutCompletion(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	go func() {
		time.Sleep(10 * time.Millisecond)
		// Close without sending turn.completed.
		close(sess.outputCh)
	}()

	ch, err := client.Stream(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{{Role: codex.RoleUser, Content: "test"}},
	})
	require.NoError(t, err)

	var chunks []codex.StreamChunk
	for chunk := range ch {
		chunks = append(chunks, chunk)
	}

	// Should get an error chunk about session ending prematurely.
	require.True(t, len(chunks) >= 1)
	lastChunk := chunks[len(chunks)-1]
	assert.True(t, lastChunk.Done)
	assert.Error(t, lastChunk.Error)
	assert.Contains(t, lastChunk.Error.Error(), "session ended without turn completion")
}

// =============================================================================
// Steer Tests
// =============================================================================

func TestSessionClient_Steer(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	err := client.Steer(context.Background(), "change direction")
	require.NoError(t, err)

	sess.closeMu.Lock()
	defer sess.closeMu.Unlock()
	require.Len(t, sess.steerCalls, 1)
	assert.Equal(t, "change direction", sess.steerCalls[0].Content)
}

// =============================================================================
// Close Tests
// =============================================================================

func TestSessionClient_Close_OwnedSession(t *testing.T) {
	mgr := newTestSessionManager()
	ctx := context.Background()

	client, err := codex.NewSessionClientWithManager(ctx, mgr, "")
	require.NoError(t, err)

	sessionID := client.SessionID()

	err = client.Close()
	assert.NoError(t, err)

	// Verify the session was closed.
	mgr.mu.RLock()
	sess, ok := mgr.sessions[sessionID]
	mgr.mu.RUnlock()

	if ok {
		assert.True(t, sess.closed, "owned session should be closed")
	}
}

func TestSessionClient_Close_UnownedSession_DoesNotClose(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	err := client.Close()
	assert.NoError(t, err)
	assert.False(t, sess.closed, "unowned session should not be closed")
}

// =============================================================================
// NewSessionClientWithManager Tests
// =============================================================================

func TestNewSessionClientWithManager_CreatesNewSession(t *testing.T) {
	mgr := newTestSessionManager()

	client, err := codex.NewSessionClientWithManager(context.Background(), mgr, "")
	require.NoError(t, err)
	require.NotNil(t, client)

	assert.NotEmpty(t, client.SessionID())
}

func TestNewSessionClientWithManager_ResumesExisting(t *testing.T) {
	mgr := newTestSessionManager()

	// Pre-populate a session.
	existing := newTestSession("existing-thread")
	mgr.addSession(existing)

	client, err := codex.NewSessionClientWithManager(context.Background(), mgr, "existing-thread")
	require.NoError(t, err)

	assert.Equal(t, "existing-thread", client.SessionID())
}

func TestNewSessionClientWithManager_CreateError(t *testing.T) {
	mgr := newTestSessionManager()
	mgr.createError = fmt.Errorf("no capacity")

	_, err := codex.NewSessionClientWithManager(context.Background(), mgr, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create session")
}

// =============================================================================
// Info and Status Tests
// =============================================================================

func TestSessionClient_Info(t *testing.T) {
	sess := newTestSession("sess-1")
	sess.info.Model = "o4-mini"
	client := codex.NewSessionClient(sess)

	info := client.Info()
	assert.Equal(t, "sess-1", info.ID)
	assert.Equal(t, "o4-mini", info.Model)
}

func TestSessionClient_Status(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	assert.Equal(t, session.StatusActive, client.Status())
}

// =============================================================================
// messagesToPrompt Tests (via Complete)
// =============================================================================

func TestSessionClient_Complete_UserMessage(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	go func() {
		time.Sleep(10 * time.Millisecond)
		sess.simulateOutput(session.OutputMessage{
			Type: codexcontract.EventTurnCompleted,
		})
	}()

	_, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{
			{Role: codex.RoleUser, Content: "Hello, Codex!"},
		},
	})
	require.NoError(t, err)

	// Verify the session received the correct prompt.
	sess.closeMu.Lock()
	defer sess.closeMu.Unlock()
	require.Len(t, sess.sendMessages, 1)
	assert.Equal(t, "Hello, Codex!", sess.sendMessages[0].Content)
}

func TestSessionClient_Complete_SystemAndUserMessages(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	go func() {
		time.Sleep(10 * time.Millisecond)
		sess.simulateOutput(session.OutputMessage{
			Type: codexcontract.EventTurnCompleted,
		})
	}()

	_, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{
			{Role: codex.RoleSystem, Content: "You are a helper."},
			{Role: codex.RoleUser, Content: "Hi!"},
		},
	})
	require.NoError(t, err)

	sess.closeMu.Lock()
	defer sess.closeMu.Unlock()
	require.Len(t, sess.sendMessages, 1)
	// The prompt should contain both system and user content.
	assert.Contains(t, sess.sendMessages[0].Content, "You are a helper.")
	assert.Contains(t, sess.sendMessages[0].Content, "Hi!")
}

func TestSessionClient_Complete_AssistantMessage(t *testing.T) {
	sess := newTestSession("sess-1")
	client := codex.NewSessionClient(sess)

	go func() {
		time.Sleep(10 * time.Millisecond)
		sess.simulateOutput(session.OutputMessage{
			Type: codexcontract.EventTurnCompleted,
		})
	}()

	_, err := client.Complete(context.Background(), codex.CompletionRequest{
		Messages: []codex.Message{
			{Role: codex.RoleUser, Content: "What is 2+2?"},
			{Role: codex.RoleAssistant, Content: "4"},
			{Role: codex.RoleUser, Content: "And 3+3?"},
		},
	})
	require.NoError(t, err)

	sess.closeMu.Lock()
	defer sess.closeMu.Unlock()
	require.Len(t, sess.sendMessages, 1)
	// Should include user content.
	assert.Contains(t, sess.sendMessages[0].Content, "What is 2+2?")
	assert.Contains(t, sess.sendMessages[0].Content, "And 3+3?")
}
