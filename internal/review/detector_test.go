package review

import (
	"context"
	"reflect"
	"testing"
)

func TestClaudeDetector_FiltersInvalidModes(t *testing.T) {
	d := NewClaudeDetector(func(ctx context.Context, diff string) (*DetectionResult, error) {
		return &DetectionResult{Modes: []Mode{ModeSecurity, Mode("bogus"), ModeStyle}, Reasoning: "ok"}, nil
	})

	modes, reasoning, err := d.Detect(context.Background(), "diff")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if reasoning != "ok" {
		t.Fatalf("expected reasoning %q, got %q", "ok", reasoning)
	}

	want := []Mode{ModeSecurity, ModeStyle}
	if !sameModeSet(modes, want) {
		t.Fatalf("expected modes %v, got %v", want, modes)
	}
}

func TestClaudeDetector_NoValidModesFallsBackToAll(t *testing.T) {
	d := NewClaudeDetector(func(ctx context.Context, diff string) (*DetectionResult, error) {
		return &DetectionResult{Modes: []Mode{Mode("bogus")}, Reasoning: "ignored"}, nil
	})

	modes, reasoning, err := d.Detect(context.Background(), "diff")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if reasoning != "No valid modes detected, running all modes" {
		t.Fatalf("unexpected reasoning: %q", reasoning)
	}
	if !reflect.DeepEqual(modes, AllModes()) {
		t.Fatalf("expected all modes, got %v", modes)
	}
}

func TestClaudeDetector_ErrorFallsBackToAll(t *testing.T) {
	d := NewClaudeDetector(func(ctx context.Context, diff string) (*DetectionResult, error) {
		return nil, context.Canceled
	})

	modes, reasoning, err := d.Detect(context.Background(), "diff")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if reasoning != "Auto-detection failed, running all modes" {
		t.Fatalf("unexpected reasoning: %q", reasoning)
	}
	if !reflect.DeepEqual(modes, AllModes()) {
		t.Fatalf("expected all modes, got %v", modes)
	}
}

func TestHeuristicDetector_DetectsSecurityKeyword(t *testing.T) {
	d := NewHeuristicDetector()
	diff := "password = input\nquery := \"select * from users\"\n" // should trigger security
	modes, _, err := d.Detect(context.Background(), diff)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !containsMode(modes, ModeSecurity) {
		t.Fatalf("expected modes to contain %q, got %v", ModeSecurity, modes)
	}
}

func TestFilterModes_ExplicitEnabledOverridesDetected(t *testing.T) {
	detected := []Mode{ModeSecurity, ModeDocs}
	enabled := map[Mode]bool{ModePerformance: true}
	disabled := map[Mode]bool{}

	modes := FilterModes(detected, enabled, disabled)
	if !sameModeSet(modes, []Mode{ModePerformance}) {
		t.Fatalf("expected only performance mode, got %v", modes)
	}
}

func TestFilterModes_DisabledFiltersDetected(t *testing.T) {
	detected := []Mode{ModeSecurity, ModeDocs}
	enabled := map[Mode]bool{}
	disabled := map[Mode]bool{ModeDocs: true}

	modes := FilterModes(detected, enabled, disabled)
	if !sameModeSet(modes, []Mode{ModeSecurity}) {
		t.Fatalf("expected only security mode, got %v", modes)
	}
}

func containsMode(modes []Mode, want Mode) bool {
	for _, m := range modes {
		if m == want {
			return true
		}
	}
	return false
}

func sameModeSet(a, b []Mode) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[Mode]int, len(a))
	for _, m := range a {
		set[m]++
	}
	for _, m := range b {
		set[m]--
		if set[m] < 0 {
			return false
		}
	}
	for _, v := range set {
		if v != 0 {
			return false
		}
	}
	return true
}
