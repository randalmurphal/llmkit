package codex_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/randalmurphal/llmkit/v2/codex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamAccumulator(t *testing.T) {
	acc := codex.NewStreamAccumulator()
	require.NotNil(t, acc)

	assert.Equal(t, "", acc.Content())
	assert.Nil(t, acc.Usage())
	assert.Equal(t, "", acc.SessionID())
	assert.False(t, acc.Done())
	assert.NoError(t, acc.Error())
}

func TestStreamAccumulator_Append_Content(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	acc.Append(codex.StreamChunk{Content: "Hello "})
	acc.Append(codex.StreamChunk{Content: "World"})

	assert.Equal(t, "Hello World", acc.Content())
}

func TestStreamAccumulator_Append_FinalContentOverrides(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	// Accumulate streamed deltas.
	acc.Append(codex.StreamChunk{Content: "partial "})
	acc.Append(codex.StreamChunk{Content: "content"})
	assert.Equal(t, "partial content", acc.Content())

	// FinalContent replaces accumulated content.
	acc.Append(codex.StreamChunk{FinalContent: "authoritative final text"})
	assert.Equal(t, "authoritative final text", acc.Content())
}

func TestStreamAccumulator_Append_Usage(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	// Usage is nil before any usage chunk arrives.
	assert.Nil(t, acc.Usage())

	acc.Append(codex.StreamChunk{Content: "text"})
	assert.Nil(t, acc.Usage(), "content-only chunk should not set usage")

	usage := &codex.TokenUsage{
		InputTokens:  100,
		OutputTokens: 50,
		TotalTokens:  150,
	}
	acc.Append(codex.StreamChunk{Usage: usage})
	require.NotNil(t, acc.Usage())
	assert.Equal(t, 100, acc.Usage().InputTokens)
	assert.Equal(t, 50, acc.Usage().OutputTokens)
	assert.Equal(t, 150, acc.Usage().TotalTokens)
}

func TestStreamAccumulator_Append_SessionID(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	acc.Append(codex.StreamChunk{SessionID: "sess-1", Content: "a"})
	assert.Equal(t, "sess-1", acc.SessionID())

	// First session ID wins; subsequent ones don't overwrite.
	acc.Append(codex.StreamChunk{SessionID: "sess-2", Content: "b"})
	assert.Equal(t, "sess-1", acc.SessionID())
}

func TestStreamAccumulator_Append_SessionID_SkipsEmpty(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	// Empty session ID should not be captured.
	acc.Append(codex.StreamChunk{Content: "text"})
	assert.Equal(t, "", acc.SessionID())

	acc.Append(codex.StreamChunk{SessionID: "real-id"})
	assert.Equal(t, "real-id", acc.SessionID())
}

func TestStreamAccumulator_Append_Error(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	assert.NoError(t, acc.Error())

	testErr := errors.New("stream failed")
	acc.Append(codex.StreamChunk{Error: testErr})
	assert.ErrorIs(t, acc.Error(), testErr)
}

func TestStreamAccumulator_Append_Done(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	assert.False(t, acc.Done())

	acc.Append(codex.StreamChunk{Content: "in progress"})
	assert.False(t, acc.Done())

	acc.Append(codex.StreamChunk{Done: true})
	assert.True(t, acc.Done())
}

func TestStreamAccumulator_ConsumeStream(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	ch := make(chan codex.StreamChunk)
	go func() {
		ch <- codex.StreamChunk{Content: "Hello ", SessionID: "s1"}
		ch <- codex.StreamChunk{Content: "World"}
		ch <- codex.StreamChunk{Done: true, Usage: &codex.TokenUsage{TotalTokens: 10}}
		close(ch)
	}()

	err := acc.ConsumeStream(ch)
	assert.NoError(t, err)
	assert.Equal(t, "Hello World", acc.Content())
	assert.Equal(t, "s1", acc.SessionID())
	assert.True(t, acc.Done())
	require.NotNil(t, acc.Usage())
	assert.Equal(t, 10, acc.Usage().TotalTokens)
}

func TestStreamAccumulator_ConsumeStream_StopsOnError(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	testErr := errors.New("server error")

	ch := make(chan codex.StreamChunk)
	go func() {
		ch <- codex.StreamChunk{Content: "partial"}
		ch <- codex.StreamChunk{Error: testErr}
		// This chunk should never be consumed because ConsumeStream
		// returns on error. We close to unblock the range.
		close(ch)
	}()

	err := acc.ConsumeStream(ch)
	assert.ErrorIs(t, err, testErr)
	assert.Equal(t, "partial", acc.Content())
	assert.ErrorIs(t, acc.Error(), testErr)
}

func TestStreamAccumulator_ConsumeStream_EmptyChannel(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	ch := make(chan codex.StreamChunk)
	close(ch)

	err := acc.ConsumeStream(ch)
	assert.NoError(t, err)
	assert.Equal(t, "", acc.Content())
}

func TestStreamAccumulator_ConsumeStreamWithCallback(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	ch := make(chan codex.StreamChunk)
	go func() {
		ch <- codex.StreamChunk{Content: "A"}
		ch <- codex.StreamChunk{Content: "B"}
		ch <- codex.StreamChunk{Content: "C"}
		ch <- codex.StreamChunk{Done: true}
		close(ch)
	}()

	var received []string
	err := acc.ConsumeStreamWithCallback(ch, func(chunk codex.StreamChunk) bool {
		received = append(received, chunk.Content)
		return true // keep consuming
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"A", "B", "C", ""}, received)
	assert.True(t, acc.Done())
}

func TestStreamAccumulator_ConsumeStreamWithCallback_StopsEarly(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	ch := make(chan codex.StreamChunk, 4)
	ch <- codex.StreamChunk{Content: "first"}
	ch <- codex.StreamChunk{Content: "second"}
	ch <- codex.StreamChunk{Content: "third"}
	ch <- codex.StreamChunk{Done: true}
	close(ch)

	callCount := 0
	err := acc.ConsumeStreamWithCallback(ch, func(chunk codex.StreamChunk) bool {
		callCount++
		return callCount < 2 // stop after 2nd call
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
	// Only first two chunks should be accumulated.
	assert.Equal(t, "firstsecond", acc.Content())
}

func TestStreamAccumulator_ConsumeStreamWithCallback_ErrorStopsBeforeCallback(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	testErr := errors.New("bad chunk")
	ch := make(chan codex.StreamChunk, 2)
	ch <- codex.StreamChunk{Content: "ok"}
	ch <- codex.StreamChunk{Error: testErr}
	close(ch)

	callbackCalled := 0
	err := acc.ConsumeStreamWithCallback(ch, func(chunk codex.StreamChunk) bool {
		callbackCalled++
		return true
	})

	assert.ErrorIs(t, err, testErr)
	// The callback should be called for the first chunk only.
	// The error chunk is appended but the callback is NOT called for it.
	assert.Equal(t, 1, callbackCalled)
}

func TestStreamAccumulator_ToResponse(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	acc.Append(codex.StreamChunk{SessionID: "sess-abc", Content: " Hello "})
	acc.Append(codex.StreamChunk{
		Content: "World  ",
		Usage: &codex.TokenUsage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
	})
	acc.Append(codex.StreamChunk{Done: true})

	resp := acc.ToResponse(2 * time.Second)

	assert.Equal(t, "Hello World", resp.Content, "content should be trimmed")
	assert.Equal(t, "sess-abc", resp.SessionID)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, 2*time.Second, resp.Duration)
	assert.Equal(t, 100, resp.Usage.InputTokens)
	assert.Equal(t, 50, resp.Usage.OutputTokens)
	assert.Equal(t, 150, resp.Usage.TotalTokens)
}

func TestStreamAccumulator_ToResponse_NilUsage(t *testing.T) {
	acc := codex.NewStreamAccumulator()
	acc.Append(codex.StreamChunk{Content: "text", Done: true})

	resp := acc.ToResponse(time.Millisecond)

	assert.Equal(t, "text", resp.Content)
	assert.Equal(t, "stop", resp.FinishReason)
	assert.Equal(t, codex.TokenUsage{}, resp.Usage, "nil usage should result in zero-value struct")
}

func TestStreamAccumulator_ToResponse_Error(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	acc.Append(codex.StreamChunk{Content: "partial"})
	acc.Append(codex.StreamChunk{Error: errors.New("failed"), Done: true})

	resp := acc.ToResponse(time.Second)

	assert.Equal(t, "error", resp.FinishReason)
	assert.Equal(t, "partial", resp.Content)
}

func TestStreamAccumulator_ConcurrentAccess(t *testing.T) {
	acc := codex.NewStreamAccumulator()

	var wg sync.WaitGroup

	// Writer goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			acc.Append(codex.StreamChunk{Content: "x"})
		}
	}()

	// Reader goroutine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			_ = acc.Content()
			_ = acc.Usage()
			_ = acc.SessionID()
			_ = acc.Done()
			_ = acc.Error()
		}
	}()

	wg.Wait()
	// No race detector failures is the assertion.
}
