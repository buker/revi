package ai

import (
	"github.com/buker/revi/internal/review"
)

// StreamContent represents streaming content from the API for progressive TUI updates.
type StreamContent struct {
	Mode    review.Mode // The review mode this content belongs to (empty for detect/commit)
	Content string      // The chunk of content received from the stream
}

// sendStreamContent safely sends stream content via a callback if provided.
// If callback is nil, this is a no-op.
func sendStreamContent(callback StreamCallback, mode review.Mode, content string) {
	if callback == nil {
		return
	}
	callback(StreamContent{
		Mode:    mode,
		Content: content,
	})
}
