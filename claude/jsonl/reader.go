// Package jsonl provides reading and tailing of Claude Code session JSONL files.
//
// Claude Code writes session history to JSONL files at:
//
//	~/.claude/projects/{normalized-path}/{sessionId}.jsonl
//
// Each line is a JSON object representing a message (user, assistant, queue-operation).
// This package provides efficient reading of these files, including real-time tailing
// for monitoring active sessions.
package jsonl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/randalmurphal/llmkit/claude/session"
)

// Reader reads JSONL session files from Claude Code.
type Reader struct {
	path string
	file *os.File
}

// NewReader creates a new JSONL reader for the given file path.
func NewReader(path string) (*Reader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open jsonl file: %w", err)
	}
	return &Reader{path: path, file: file}, nil
}

// Path returns the file path being read.
func (r *Reader) Path() string {
	return r.path
}

// Close closes the underlying file.
func (r *Reader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// ReadAll reads all messages from the JSONL file.
func (r *Reader) ReadAll() ([]session.JSONLMessage, error) {
	// Seek to beginning
	if _, err := r.file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek to start: %w", err)
	}

	var messages []session.JSONLMessage
	scanner := bufio.NewScanner(r.file)
	// Increase buffer for large messages
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		msg, err := session.ParseJSONLMessage(line)
		if err != nil {
			// Skip malformed lines
			continue
		}
		messages = append(messages, *msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan jsonl: %w", err)
	}

	return messages, nil
}

// ReadFrom reads all messages starting from a specific byte offset.
// Returns the new offset after reading.
func (r *Reader) ReadFrom(offset int64) ([]session.JSONLMessage, int64, error) {
	if _, err := r.file.Seek(offset, io.SeekStart); err != nil {
		return nil, offset, fmt.Errorf("seek to offset: %w", err)
	}

	var messages []session.JSONLMessage
	scanner := bufio.NewScanner(r.file)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		offset += int64(len(line)) + 1 // +1 for newline
		if len(line) == 0 {
			continue
		}

		msg, err := session.ParseJSONLMessage(line)
		if err != nil {
			continue
		}
		messages = append(messages, *msg)
	}

	if err := scanner.Err(); err != nil {
		return nil, offset, fmt.Errorf("scan jsonl: %w", err)
	}

	return messages, offset, nil
}

// Tail follows the JSONL file and sends new messages to the returned channel.
// The channel is closed when the context is cancelled or an unrecoverable error occurs.
// Uses fsnotify for efficient file watching with polling fallback.
func (r *Reader) Tail(ctx context.Context) <-chan session.JSONLMessage {
	ch := make(chan session.JSONLMessage, 100)

	go func() {
		defer close(ch)

		// Seek to end to only show new content
		offset, err := r.file.Seek(0, io.SeekEnd)
		if err != nil {
			return
		}

		// Try fsnotify first
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			// Fallback to polling
			r.tailPolling(ctx, ch, offset)
			return
		}
		defer watcher.Close()

		// Watch the directory (more reliable than watching file directly)
		dir := filepath.Dir(r.path)
		if err := watcher.Add(dir); err != nil {
			watcher.Close()
			r.tailPolling(ctx, ch, offset)
			return
		}

		r.tailWithWatcher(ctx, ch, watcher, offset)
	}()

	return ch
}

// tailWithWatcher uses fsnotify for efficient file watching.
func (r *Reader) tailWithWatcher(ctx context.Context, ch chan<- session.JSONLMessage, watcher *fsnotify.Watcher, offset int64) {
	baseName := filepath.Base(r.path)
	reader := bufio.NewReader(r.file)

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Only care about writes to our file
			if filepath.Base(event.Name) != baseName {
				continue
			}
			if !event.Has(fsnotify.Write) {
				continue
			}

			// Check for truncation
			info, err := r.file.Stat()
			if err != nil {
				continue
			}
			if info.Size() < offset {
				// File truncated, reset
				r.file.Seek(0, io.SeekStart)
				offset = 0
				reader.Reset(r.file)
			}

			// Read new messages
			offset = r.readNewMessages(reader, ch, offset)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			// Log but continue - usually recoverable
			_ = err
		}
	}
}

// tailPolling uses polling as a fallback when fsnotify isn't available.
func (r *Reader) tailPolling(ctx context.Context, ch chan<- session.JSONLMessage, offset int64) {
	reader := bufio.NewReader(r.file)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			// Check for truncation
			info, err := r.file.Stat()
			if err != nil {
				continue
			}
			if info.Size() < offset {
				r.file.Seek(0, io.SeekStart)
				offset = 0
				reader.Reset(r.file)
			}

			// Read new messages
			offset = r.readNewMessages(reader, ch, offset)
		}
	}
}

// readNewMessages reads available messages from the reader.
func (r *Reader) readNewMessages(reader *bufio.Reader, ch chan<- session.JSONLMessage, offset int64) int64 {
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			offset += int64(len(line))
			// Trim newline for parsing
			line = []byte(strings.TrimSuffix(string(line), "\n"))
			if len(line) > 0 {
				if msg, err := session.ParseJSONLMessage(line); err == nil {
					select {
					case ch <- *msg:
					default:
						// Channel full, skip
					}
				}
			}
		}
		if err != nil {
			break
		}
	}
	return offset
}

// ReadFile reads all messages from a JSONL file path.
// Convenience function that opens, reads, and closes the file.
func ReadFile(path string) ([]session.JSONLMessage, error) {
	r, err := NewReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return r.ReadAll()
}

// FindSessionFiles returns all JSONL files in a Claude projects directory.
// projectsDir should be ~/.claude/projects/
func FindSessionFiles(projectsDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() && strings.HasSuffix(path, ".jsonl") {
			files = append(files, path)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk projects dir: %w", err)
	}

	return files, nil
}

// Summary contains aggregate statistics from a JSONL session file.
type Summary struct {
	SessionID    string
	MessageCount int
	UserMessages int
	AssistantMessages int
	TotalInputTokens  int
	TotalOutputTokens int
	TotalCacheCreationTokens int
	TotalCacheReadTokens int
	Models       map[string]int // Model name -> message count
	ToolCalls    int
	FirstTimestamp string
	LastTimestamp  string
}

// Summarize reads a JSONL file and returns aggregate statistics.
func Summarize(path string) (*Summary, error) {
	messages, err := ReadFile(path)
	if err != nil {
		return nil, err
	}

	summary := &Summary{
		Models: make(map[string]int),
	}

	for _, msg := range messages {
		summary.MessageCount++

		if summary.FirstTimestamp == "" {
			summary.FirstTimestamp = msg.Timestamp
		}
		summary.LastTimestamp = msg.Timestamp

		if summary.SessionID == "" && msg.SessionID != "" {
			summary.SessionID = msg.SessionID
		}

		if msg.IsUser() {
			summary.UserMessages++
		}
		if msg.IsAssistant() {
			summary.AssistantMessages++

			if model := msg.GetModel(); model != "" {
				summary.Models[model]++
			}

			if usage := msg.GetUsage(); usage != nil {
				summary.TotalInputTokens += usage.InputTokens
				summary.TotalOutputTokens += usage.OutputTokens
				summary.TotalCacheCreationTokens += usage.CacheCreationInputTokens
				summary.TotalCacheReadTokens += usage.CacheReadInputTokens
			}

			summary.ToolCalls += len(msg.GetToolCalls())
		}
	}

	return summary, nil
}

// ExtractTodos extracts all TodoWrite snapshots from a JSONL file.
// Returns a slice of todo lists in chronological order.
func ExtractTodos(path string) ([][]session.TodoItem, error) {
	messages, err := ReadFile(path)
	if err != nil {
		return nil, err
	}

	var todoSnapshots [][]session.TodoItem

	for _, msg := range messages {
		if msg.HasTodoUpdate() {
			todos := msg.GetTodos()
			if len(todos) > 0 {
				todoSnapshots = append(todoSnapshots, todos)
			}
		}
	}

	return todoSnapshots, nil
}

// ExtractToolCalls extracts all tool calls from a JSONL file.
// Returns a slice of tool call content blocks.
func ExtractToolCalls(path string) ([]session.JSONLContentBlock, error) {
	messages, err := ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tools []session.JSONLContentBlock

	for _, msg := range messages {
		tools = append(tools, msg.GetToolCalls()...)
	}

	return tools, nil
}

// FilterByModel returns only messages from a specific model.
func FilterByModel(messages []session.JSONLMessage, model string) []session.JSONLMessage {
	var filtered []session.JSONLMessage
	for _, msg := range messages {
		if msg.GetModel() == model {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

// ToJSON converts messages to a JSON array for export.
func ToJSON(messages []session.JSONLMessage) ([]byte, error) {
	return json.MarshalIndent(messages, "", "  ")
}
