package ai

import (
	"context"
	"errors"
	"testing"

	claudecode "github.com/rokrokss/claude-code-sdk-go"
)

// mockTransport implements claudecode.Transport for testing
type mockTransport struct {
	connectCalled    bool
	connectErr       error
	closeCalled      bool
	closeErr         error
	msgChan          chan claudecode.Message
	errChan          chan error
	sendMessageErr   error
	messagesReceived []claudecode.StreamMessage
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		msgChan: make(chan claudecode.Message, 10),
		errChan: make(chan error, 1),
	}
}

func (m *mockTransport) Connect(ctx context.Context) error {
	m.connectCalled = true
	return m.connectErr
}

func (m *mockTransport) SendMessage(ctx context.Context, msg claudecode.StreamMessage) error {
	m.messagesReceived = append(m.messagesReceived, msg)
	return m.sendMessageErr
}

func (m *mockTransport) ReceiveMessages(ctx context.Context) (<-chan claudecode.Message, <-chan error) {
	return m.msgChan, m.errChan
}

func (m *mockTransport) Interrupt(ctx context.Context) error {
	return nil
}

func (m *mockTransport) Close() error {
	m.closeCalled = true
	return m.closeErr
}

// TestWithClientPattern_ConnectsAndDisconnects verifies the WithClient pattern
// properly establishes and cleans up the client connection.
func TestWithClientPattern_ConnectsAndDisconnects(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	var clientUsed bool
	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		clientUsed = true
		return nil
	})

	if err != nil {
		t.Fatalf("WithClientTransport() error = %v, want nil", err)
	}
	if !clientUsed {
		t.Error("callback function was not executed")
	}
	if !transport.connectCalled {
		t.Error("Connect() was not called")
	}
	if !transport.closeCalled {
		t.Error("Close() was not called for cleanup")
	}
}

// TestWithClientPattern_CleanupOnError verifies that the client properly
// cleans up resources even when the callback returns an error.
func TestWithClientPattern_CleanupOnError(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()
	expectedErr := errors.New("callback error")

	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		return expectedErr
	})

	if err == nil {
		t.Fatal("WithClientTransport() error = nil, want error")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("WithClientTransport() error = %v, want %v", err, expectedErr)
	}
	if !transport.closeCalled {
		t.Error("Close() was not called despite callback error - resource leak")
	}
}

// TestWithClientPattern_ClientReuseWithinCallback verifies that the same client
// instance can be used for multiple operations within a single WithClient call.
func TestWithClientPattern_ClientReuseWithinCallback(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	// Pre-populate response messages
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: "response 1"},
		},
	}
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: "response 2"},
		},
	}
	close(transport.msgChan)

	var queriesSent int
	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		// First query
		if err := client.Query(ctx, "query 1"); err != nil {
			return err
		}
		queriesSent++

		// Second query with same client
		if err := client.Query(ctx, "query 2"); err != nil {
			return err
		}
		queriesSent++

		return nil
	})

	if err != nil {
		t.Fatalf("WithClientTransport() error = %v, want nil", err)
	}
	if queriesSent != 2 {
		t.Errorf("queries sent = %d, want 2", queriesSent)
	}
	// Verify both queries went through the same transport (same connection)
	if len(transport.messagesReceived) != 2 {
		t.Errorf("messages received by transport = %d, want 2", len(transport.messagesReceived))
	}
}

// TestWithClientPattern_ConnectionError verifies proper error handling when
// the CLI is not available or connection fails.
func TestWithClientPattern_ConnectionError(t *testing.T) {
	transport := newMockTransport()
	transport.connectErr = errors.New("CLI not found: claude command not found in PATH")
	ctx := context.Background()

	var callbackExecuted bool
	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		callbackExecuted = true
		return nil
	})

	if err == nil {
		t.Fatal("WithClientTransport() error = nil, want connection error")
	}
	if callbackExecuted {
		t.Error("callback was executed despite connection failure")
	}
	// Close should not be called if Connect failed
	if transport.closeCalled {
		t.Error("Close() was called despite connection failure")
	}
}

// TestNewClientWrapper_StoresModel verifies that our wrapper properly stores
// the model configuration for use in queries.
func TestNewClientWrapper_StoresModel(t *testing.T) {
	model := "claude-sonnet-4-20250514"
	wrapper := NewClientWrapper(model)

	if wrapper.model != model {
		t.Errorf("model = %q, want %q", wrapper.model, model)
	}
}
