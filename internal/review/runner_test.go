package review

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestRunner_RunPreservesModeOrder(t *testing.T) {
	modes := []Mode{ModeSecurity, ModeDocs, ModeErrors}

	sleepByMode := map[Mode]time.Duration{
		ModeSecurity: 30 * time.Millisecond,
		ModeDocs:     10 * time.Millisecond,
		ModeErrors:   20 * time.Millisecond,
	}

	runner := NewRunner(
		func(ctx context.Context, mode Mode, diff string) (*Result, error) {
			time.Sleep(sleepByMode[mode])
			return &Result{Mode: mode, Status: StatusNoIssues}, nil
		},
		nil,
	)

	results := runner.Run(context.Background(), modes, "diff")
	if len(results) != len(modes) {
		t.Fatalf("expected %d results, got %d", len(modes), len(results))
	}
	for i, mode := range modes {
		if results[i] == nil {
			t.Fatalf("expected non-nil result at %d", i)
		}
		if results[i].Mode != mode {
			t.Fatalf("expected result[%d].Mode = %q, got %q", i, mode, results[i].Mode)
		}
	}
}

func TestRunner_StatusCallbackTransitions(t *testing.T) {
	modes := []Mode{ModeSecurity, ModeErrors}

	var mu sync.Mutex
	events := make(map[Mode][]Status)
	cb := func(mode Mode, status Status) {
		mu.Lock()
		defer mu.Unlock()
		events[mode] = append(events[mode], status)
	}

	runner := NewRunner(
		func(ctx context.Context, mode Mode, diff string) (*Result, error) {
			if mode == ModeErrors {
				return nil, context.Canceled
			}
			return &Result{Mode: mode, Status: StatusNoIssues}, nil
		},
		cb,
	)

	_ = runner.Run(context.Background(), modes, "diff")

	mu.Lock()
	defer mu.Unlock()

	for _, mode := range modes {
		seq := events[mode]
		if len(seq) != 2 {
			t.Fatalf("expected 2 status callbacks for %q, got %v", mode, seq)
		}
		if seq[0] != StatusRunning {
			t.Fatalf("expected first status for %q to be %q, got %q", mode, StatusRunning, seq[0])
		}
		wantEnd := StatusDone
		if mode == ModeErrors {
			wantEnd = StatusFailed
		}
		if seq[1] != wantEnd {
			t.Fatalf("expected final status for %q to be %q, got %q", mode, wantEnd, seq[1])
		}
	}
}

func TestSummarize_ShouldBlock_GetBlockReason(t *testing.T) {
	results := []*Result{
		nil,
		{Mode: ModeSecurity, Status: StatusFailed, Error: "boom"},
		{Mode: ModeStyle, Status: StatusNoIssues, Issues: nil},
		{
			Mode:   ModeErrors,
			Status: StatusIssues,
			Issues: []Issue{{Severity: "high", Description: "bad"}, {Severity: "low", Description: "meh"}},
		},
		{
			Mode:   ModeDocs,
			Status: StatusIssues,
			Issues: []Issue{{Severity: "medium", Description: "ok"}},
		},
	}

	summary := Summarize(results)
	if summary.TotalReviews != len(results) {
		t.Fatalf("expected TotalReviews %d, got %d", len(results), summary.TotalReviews)
	}
	if summary.FailedReviews != 1 {
		t.Fatalf("expected FailedReviews 1, got %d", summary.FailedReviews)
	}
	if summary.IssuesFound != 3 {
		t.Fatalf("expected IssuesFound 3, got %d", summary.IssuesFound)
	}
	if summary.HighSeverity != 1 || summary.MediumSeverity != 1 || summary.LowSeverity != 1 {
		t.Fatalf("unexpected severity counts: high=%d medium=%d low=%d", summary.HighSeverity, summary.MediumSeverity, summary.LowSeverity)
	}

	if !ShouldBlock(results, true) {
		t.Fatal("expected ShouldBlock to be true when high severity issues exist")
	}
	if ShouldBlock(results, false) {
		t.Fatal("expected ShouldBlock to be false when blocking disabled")
	}

	reason := GetBlockReason(results)
	if reason != "1 high-severity issue found" {
		t.Fatalf("expected block reason %q, got %q", "1 high-severity issue found", reason)
	}
}
