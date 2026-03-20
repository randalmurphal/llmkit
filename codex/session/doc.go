// Package session provides long-running Codex CLI session management
// with bidirectional JSON-RPC 2.0 I/O over the app-server protocol.
//
// This package enables managing persistent Codex CLI processes that
// communicate via JSON-RPC 2.0 over stdio, allowing for real-time
// bidirectional conversation without spawning a new process for each message.
//
// # Basic Usage
//
// Create a session manager and start a session:
//
//	mgr := session.NewManager()
//	defer mgr.CloseAll()
//
//	sess, err := mgr.Create(ctx,
//	    session.WithModel("o4-mini"),
//	    session.WithWorkdir("/path/to/project"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Send a message
//	err = sess.Send(ctx, session.NewUserMessage("Hello, Codex!"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Read responses
//	for msg := range sess.Output() {
//	    if msg.IsItemUpdate() {
//	        fmt.Println(msg.GetText())
//	    }
//	    if msg.IsTurnComplete() {
//	        break
//	    }
//	}
//
// # Protocol
//
// Codex app-server speaks JSON-RPC 2.0 over newline-delimited stdio.
// Key methods:
//   - thread/start: Create a new thread (returns thread_id)
//   - thread/resume: Resume an existing thread
//   - turn/start: Send a new user message on a thread
//   - turn/steer: Inject input into an actively running turn
//   - shutdown: Gracefully terminate the server
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
// The Output channel receives notifications from the app-server:
//   - thread.started: Thread creation confirmed
//   - turn.started: A turn has begun processing
//   - turn.completed: A turn has finished
//   - turn.failed: A turn encountered an error
//   - item.started: An item (message, tool call) has begun
//   - item.updated: An item has new content
//   - item.completed: An item has finished
//
// Use the Is*() methods and GetText() to handle each notification type.
package session
