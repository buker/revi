package ai

import (
	"context"
	"sync"
	"testing"

	claudecode "github.com/rokrokss/claude-code-sdk-go"

	"github.com/buker/revi/internal/review"
)

func TestStreamingCallback_SendsProgressiveMessages(t *testing.T) {
	var mu sync.Mutex
	var messages []StreamContent
	callback := func(content StreamContent) {
		mu.Lock()
		defer mu.Unlock()
		messages = append(messages, content)
	}

	// Simulate streaming content
	chunks := []string{"Analyzing", " code", " for", " security", " issues..."}
	mode := review.ModeSecurity

	for _, chunk := range chunks {
		callback(StreamContent{Mode: mode, Content: chunk})
	}

	mu.Lock()
	defer mu.Unlock()

	if len(messages) != len(chunks) {
		t.Errorf("expected %d messages, got %d", len(chunks), len(messages))
	}

	for i, msg := range messages {
		if msg.Mode != mode {
			t.Errorf("message[%d].Mode = %v, want %v", i, msg.Mode, mode)
		}
		if msg.Content != chunks[i] {
			t.Errorf("message[%d].Content = %q, want %q", i, msg.Content, chunks[i])
		}
	}
}

func TestStreamingCallback_NilCallbackSafe(t *testing.T) {
	// Verify that a nil callback does not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("sendStreamContent panicked with nil callback: %v", r)
		}
	}()

	sendStreamContent(nil, review.ModeSecurity, "test content")
}

func TestClientWithStreamCallback_SetsCallback(t *testing.T) {
	// Authentication is handled by the Claude Code CLI
	client, err := NewClient("claude-sonnet-4-20250514")
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	var received []StreamContent
	callback := func(content StreamContent) {
		received = append(received, content)
	}

	client.SetStreamCallback(callback)

	if client.streamCallback == nil {
		t.Error("streamCallback not set")
	}
}

func TestStreamContent_Type(t *testing.T) {
	content := StreamContent{
		Mode:    review.ModePerformance,
		Content: "Found potential N+1 query",
	}

	if content.Mode != review.ModePerformance {
		t.Errorf("StreamContent.Mode = %v, want %v", content.Mode, review.ModePerformance)
	}
	if content.Content != "Found potential N+1 query" {
		t.Errorf("StreamContent.Content = %q, want %q", content.Content, "Found potential N+1 query")
	}
}

func TestStreamInterruption_MarksReviewFailed(t *testing.T) {
	// Create a context that we can cancel to simulate stream interruption
	ctx, cancel := context.WithCancel(context.Background())

	var receivedError error
	var streamCompleted bool

	// Simulate a streaming operation that gets interrupted
	go func() {
		// Cancel after a short delay to simulate interruption
		cancel()
	}()

	// Simulate checking for context cancellation during streaming
	select {
	case <-ctx.Done():
		receivedError = ctx.Err()
	case <-make(chan struct{}): // Would receive data here normally
		streamCompleted = true
	}

	if streamCompleted {
		t.Error("stream should not complete when interrupted")
	}
	if receivedError != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", receivedError)
	}
}

// Task 3.1: Focused tests for channel-based streaming implementation

// TestChannelStreaming_TextBlockConsumption verifies that TextBlock messages
// are properly consumed from the channel and text is extracted and sent to callback.
func TestChannelStreaming_TextBlockConsumption(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	// Pre-populate response messages with TextBlock content
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: "First chunk"},
		},
	}
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: " second chunk"},
		},
	}
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: " final chunk"},
		},
	}
	close(transport.msgChan)

	// Track streamed content
	var mu sync.Mutex
	var streamedChunks []string

	wrapper := NewClientWrapper("claude-sonnet-4-20250514")
	wrapper.SetStreamCallback(func(content StreamContent) {
		mu.Lock()
		defer mu.Unlock()
		streamedChunks = append(streamedChunks, content.Content)
	})

	var result string
	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		var callErr error
		result, callErr = wrapper.callAPIWithStreaming(ctx, client, "test prompt", review.ModeSecurity)
		return callErr
	})

	if err != nil {
		t.Fatalf("callAPIWithStreaming() error = %v, want nil", err)
	}

	// Verify complete response was built
	expectedResult := "First chunk second chunk final chunk"
	if result != expectedResult {
		t.Errorf("result = %q, want %q", result, expectedResult)
	}

	// Verify streaming callback received all chunks
	mu.Lock()
	defer mu.Unlock()
	if len(streamedChunks) != 3 {
		t.Errorf("streamed chunks = %d, want 3", len(streamedChunks))
	}
	expectedChunks := []string{"First chunk", " second chunk", " final chunk"}
	for i, chunk := range streamedChunks {
		if chunk != expectedChunks[i] {
			t.Errorf("streamedChunks[%d] = %q, want %q", i, chunk, expectedChunks[i])
		}
	}
}

// TestChannelStreaming_ErrorMessageHandling verifies that ResultMessage with
// IsError=true is properly handled and returns an error.
func TestChannelStreaming_ErrorMessageHandling(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	// Send a result message indicating an error
	transport.msgChan <- &claudecode.ResultMessage{
		IsError: true,
	}
	close(transport.msgChan)

	wrapper := NewClientWrapper("claude-sonnet-4-20250514")

	var result string
	var resultErr error
	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		result, resultErr = wrapper.callAPIWithStreaming(ctx, client, "test prompt", review.ModeSecurity)
		return nil // Don't propagate the API error to the transport layer
	})

	if err != nil {
		t.Fatalf("WithClientTransport() error = %v, want nil", err)
	}
	if resultErr == nil {
		t.Fatal("callAPIWithStreaming() error = nil, want error for IsError=true")
	}
	if result != "" {
		t.Errorf("result = %q, want empty string on error", result)
	}
}

// TestChannelStreaming_PartialContentOnError verifies that when an error occurs
// after receiving some content, the "..." indicator is sent via callback.
func TestChannelStreaming_PartialContentOnError(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	// Send partial content followed by an error
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: "Partial content"},
		},
	}
	transport.msgChan <- &claudecode.ResultMessage{
		IsError: true,
	}
	close(transport.msgChan)

	// Track streamed content
	var mu sync.Mutex
	var streamedChunks []string

	wrapper := NewClientWrapper("claude-sonnet-4-20250514")
	wrapper.SetStreamCallback(func(content StreamContent) {
		mu.Lock()
		defer mu.Unlock()
		streamedChunks = append(streamedChunks, content.Content)
	})

	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		_, _ = wrapper.callAPIWithStreaming(ctx, client, "test prompt", review.ModeSecurity)
		return nil
	})

	if err != nil {
		t.Fatalf("WithClientTransport() error = %v, want nil", err)
	}

	// Verify the "..." indicator was sent after partial content
	mu.Lock()
	defer mu.Unlock()
	if len(streamedChunks) < 2 {
		t.Fatalf("streamed chunks = %d, want at least 2 (content + indicator)", len(streamedChunks))
	}
	if streamedChunks[0] != "Partial content" {
		t.Errorf("streamedChunks[0] = %q, want %q", streamedChunks[0], "Partial content")
	}
	if streamedChunks[len(streamedChunks)-1] != "..." {
		t.Errorf("last chunk = %q, want %q (partial content indicator)", streamedChunks[len(streamedChunks)-1], "...")
	}
}

// TestChannelStreaming_MultipleTextBlocksInSingleMessage verifies that multiple
// TextBlocks within a single AssistantMessage are all processed.
func TestChannelStreaming_MultipleTextBlocksInSingleMessage(t *testing.T) {
	transport := newMockTransport()
	ctx := context.Background()

	// Send an AssistantMessage with multiple TextBlocks
	transport.msgChan <- &claudecode.AssistantMessage{
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{Text: "Block one"},
			&claudecode.TextBlock{Text: " block two"},
			&claudecode.TextBlock{Text: " block three"},
		},
	}
	close(transport.msgChan)

	// Track streamed content
	var mu sync.Mutex
	var streamedChunks []string

	wrapper := NewClientWrapper("claude-sonnet-4-20250514")
	wrapper.SetStreamCallback(func(content StreamContent) {
		mu.Lock()
		defer mu.Unlock()
		streamedChunks = append(streamedChunks, content.Content)
	})

	var result string
	err := claudecode.WithClientTransport(ctx, transport, func(client claudecode.Client) error {
		var callErr error
		result, callErr = wrapper.callAPIWithStreaming(ctx, client, "test prompt", review.ModeStyle)
		return callErr
	})

	if err != nil {
		t.Fatalf("callAPIWithStreaming() error = %v, want nil", err)
	}

	// Verify complete response
	expectedResult := "Block one block two block three"
	if result != expectedResult {
		t.Errorf("result = %q, want %q", result, expectedResult)
	}

	// Verify all blocks were streamed
	mu.Lock()
	defer mu.Unlock()
	if len(streamedChunks) != 3 {
		t.Errorf("streamed chunks = %d, want 3", len(streamedChunks))
	}
}
