package tui

import "github.com/buker/revi/internal/review"

// MsgStreamContent is sent when streaming content is received from the AI.
// This message allows the TUI to display progressive content updates
// during review execution without disrupting the layout.
type MsgStreamContent struct {
	Mode    review.Mode // The review mode this content belongs to (empty for detect/commit)
	Content string      // The chunk of content received from the stream
}
