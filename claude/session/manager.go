package session

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// SessionManager manages multiple Claude CLI sessions.
type SessionManager interface {
	// Create starts a new Claude session.
	Create(ctx context.Context, opts ...SessionOption) (Session, error)

	// Get retrieves an existing session by ID.
	Get(sessionID string) (Session, bool)

	// Resume resumes a persisted session by ID.
	Resume(ctx context.Context, sessionID string, opts ...SessionOption) (Session, error)

	// Close closes a specific session.
	Close(sessionID string) error

	// CloseAll closes all active sessions.
	CloseAll() error

	// List returns all active session IDs.
	List() []string

	// Count returns the number of active sessions.
	Count() int

	// Info returns information about a session.
	Info(sessionID string) (*SessionInfo, bool)
}

// manager implements SessionManager.
type manager struct {
	config     managerConfig
	sessions   map[string]*session
	aliases    map[string]string
	mu         sync.RWMutex
	closed     bool
	closedOnce sync.Once
	stopClean  chan struct{}
	nextKey    atomic.Uint64
}

// NewManager creates a new session manager.
func NewManager(opts ...ManagerOption) SessionManager {
	cfg := defaultManagerConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	m := &manager{
		config:    cfg,
		sessions:  make(map[string]*session),
		aliases:   make(map[string]string),
		stopClean: make(chan struct{}),
	}

	// Start cleanup goroutine if TTL is configured
	if cfg.sessionTTL > 0 && cfg.cleanupInterval > 0 {
		go m.cleanupLoop()
	}

	return m
}

// Create implements SessionManager.
func (m *manager) Create(ctx context.Context, opts ...SessionOption) (Session, error) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, fmt.Errorf("manager is closed")
	}

	if len(m.sessions) >= m.config.maxSessions {
		m.mu.Unlock()
		return nil, fmt.Errorf("max sessions reached (%d)", m.config.maxSessions)
	}
	m.mu.Unlock()

	// Apply default options first, then user options
	allOpts := make([]SessionOption, 0, len(m.config.defaultOpts)+len(opts))
	allOpts = append(allOpts, m.config.defaultOpts...)
	allOpts = append(allOpts, opts...)

	s, err := newSession(ctx, allOpts...)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check limits after session creation
	if len(m.sessions) >= m.config.maxSessions {
		_ = s.Close() // Best effort cleanup
		return nil, fmt.Errorf("max sessions reached (%d)", m.config.maxSessions)
	}

	key := m.sessionKeyForCreate(s)
	m.sessions[key] = s
	if sessionID := s.ID(); sessionID != "" {
		if _, exists := m.aliases[sessionID]; exists {
			delete(m.sessions, key)
			_ = s.Close()
			return nil, fmt.Errorf("session already tracked: %s", sessionID)
		}
		m.aliases[sessionID] = key
	}

	// Start goroutines to track init/close lifecycle.
	go m.trackSessionID(key, s)
	go m.watchSession(key, s)

	return s, nil
}

func (m *manager) sessionKeyForCreate(s *session) string {
	if id := s.ID(); id != "" {
		return id
	}
	return fmt.Sprintf("__pending__:%d", m.nextKey.Add(1))
}

func (m *manager) trackSessionID(key string, s *session) {
	if s.ID() == "" {
		if err := s.WaitForInit(context.Background()); err != nil {
			return
		}
	}

	sessionID := s.ID()
	if sessionID == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	current, ok := m.sessions[key]
	if !ok || current != s {
		return
	}
	if existingKey, exists := m.aliases[sessionID]; exists && existingKey != key {
		return
	}
	m.aliases[sessionID] = key
}

// watchSession removes a session from the map when it closes.
func (m *manager) watchSession(key string, s *session) {
	<-s.done
	m.mu.Lock()
	delete(m.sessions, key)
	sessionID := s.ID()
	if sessionID != "" && m.aliases[sessionID] == key {
		delete(m.aliases, sessionID)
	}
	m.mu.Unlock()
}

// Get implements SessionManager.
func (m *manager) Get(sessionID string) (Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, ok := m.aliases[sessionID]
	if !ok {
		return nil, false
	}
	s, ok := m.sessions[key]
	if !ok || s.Status() != StatusActive {
		return nil, false
	}
	return s, true
}

// Resume implements SessionManager.
func (m *manager) Resume(ctx context.Context, sessionID string, opts ...SessionOption) (Session, error) {
	// Check if session is already active
	if s, ok := m.Get(sessionID); ok {
		return s, nil
	}

	// Add resume option
	resumeOpts := append([]SessionOption{WithResume(sessionID)}, opts...)
	return m.Create(ctx, resumeOpts...)
}

// Close implements SessionManager.
func (m *manager) Close(sessionID string) error {
	m.mu.Lock()
	key, ok := m.aliases[sessionID]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}
	s, ok := m.sessions[key]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}
	m.mu.Unlock()

	return s.Close()
}

// CloseAll implements SessionManager.
func (m *manager) CloseAll() error {
	m.closedOnce.Do(func() {
		close(m.stopClean)
		m.closed = true
	})

	m.mu.Lock()
	sessions := make([]*session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.mu.Unlock()

	var lastErr error
	for _, s := range sessions {
		if err := s.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// List implements SessionManager.
func (m *manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.sessions))
	for id, key := range m.aliases {
		s, ok := m.sessions[key]
		if ok && s.Status() == StatusActive {
			ids = append(ids, id)
		}
	}
	return ids
}

// Count implements SessionManager.
func (m *manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// Info implements SessionManager.
func (m *manager) Info(sessionID string) (*SessionInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, ok := m.aliases[sessionID]
	if !ok {
		return nil, false
	}
	s, ok := m.sessions[key]
	if !ok {
		return nil, false
	}

	info := s.Info()
	return &info, true
}

// cleanupLoop periodically removes expired sessions.
func (m *manager) cleanupLoop() {
	ticker := time.NewTicker(m.config.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopClean:
			return
		case <-ticker.C:
			m.cleanupExpired()
		}
	}
}

// cleanupExpired closes sessions that have been idle too long.
func (m *manager) cleanupExpired() {
	m.mu.RLock()
	var expired []*session
	cutoff := time.Now().Add(-m.config.sessionTTL)

	for _, s := range m.sessions {
		if s.lastActivity.Load().(time.Time).Before(cutoff) {
			expired = append(expired, s)
		}
	}
	m.mu.RUnlock()

	for _, s := range expired {
		_ = s.Close() // Best effort cleanup of expired sessions
	}
}
