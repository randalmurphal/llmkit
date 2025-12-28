// Package session provides long-running Claude CLI session management
// with bidirectional stream-json I/O.
//
// This package enables managing persistent Claude CLI processes that
// communicate via JSON streaming, allowing for real-time bidirectional
// conversation without spawning a new process for each message.
//
// # Basic Usage
//
// Create a session manager and start a session:
//
//	mgr := session.NewManager()
//	defer mgr.CloseAll()
//
//	sess, err := mgr.Create(ctx,
//	    session.WithModel("sonnet"),
//	    session.WithWorkdir("/path/to/project"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Send a message
//	err = sess.Send(ctx, session.NewUserMessage("Hello, Claude!"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Read responses
//	for msg := range sess.Output() {
//	    if msg.IsAssistant() {
//	        fmt.Println(msg.GetText())
//	    }
//	    if msg.IsResult() {
//	        break
//	    }
//	}
//
// # Session Lifecycle
//
// Sessions go through the following states:
//   - StatusCreating: Process is starting up
//   - StatusActive: Session is ready for messages
//   - StatusClosing: Session is shutting down
//   - StatusClosed: Session has ended
//   - StatusError: Session encountered an error
//
// # Thread Safety
//
// Both Session and SessionManager are safe for concurrent use from
// multiple goroutines.
//
// # Message Types
//
// The Output channel receives several message types:
//   - system/init: Session initialization with available tools
//   - assistant: Claude's responses
//   - result: Final result with token usage and cost
//   - system/hook_response: Hook execution output (filtered by default)
//
// Use the Is*() methods and type-specific fields to handle each type.
package session
