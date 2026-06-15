package sat

import (
	"testing"
)

func TestNewModeSwitcher(t *testing.T) {
	ms := NewModeSwitcher()
	if ms.Mode() != ModeFocused {
		t.Error("new switcher should start in focused mode")
	}
	if ms.conflictLimit != 1000 {
		t.Errorf("conflictLimit = %d, want 1000", ms.conflictLimit)
	}
}

func TestModeSwitcherFocusedToStable(t *testing.T) {
	ms := NewModeSwitcher()
	// Not enough conflicts yet
	if ms.ShouldSwitch(500, 0) {
		t.Error("should not switch at 500 conflicts")
	}
	// Exceed conflict limit
	if !ms.ShouldSwitch(1000, 0) {
		t.Error("should switch at 1000 conflicts")
	}
	ms.Switch(1000, 0)
	if ms.Mode() != ModeStable {
		t.Error("should be in stable mode after switch")
	}
}

func TestModeSwitcherStableToFocused(t *testing.T) {
	ms := NewModeSwitcher()
	// Switch to stable first
	ms.Switch(1000, 0)
	if ms.Mode() != ModeStable {
		t.Fatal("expected stable mode")
	}
	// Not enough decisions yet
	if ms.ShouldSwitch(1000, 200) {
		t.Error("should not switch at 200 decisions")
	}
	// Exceed tick limit
	if !ms.ShouldSwitch(1000, 500) {
		t.Error("should switch at 500 decisions")
	}
	ms.Switch(1000, 500)
	if ms.Mode() != ModeFocused {
		t.Error("should be back in focused mode")
	}
}

func TestModeSwitcherLimitScaling(t *testing.T) {
	ms := NewModeSwitcher()
	// First switch: focused → stable. switches=1, n=1, tickLimit = 500*1 = 500
	ms.Switch(1000, 0)
	if ms.tickLimit != 500 {
		t.Errorf("first stable tick limit = %d, want 500", ms.tickLimit)
	}
	// Second switch: stable → focused. switches=2, n=2, conflictLimit = 1000*4 = 4000
	ms.Switch(1000, 500)
	if ms.conflictLimit != 4000 {
		t.Errorf("second focused conflict limit = %d, want 4000", ms.conflictLimit)
	}
}

func TestReluctantDoublingSequence(t *testing.T) {
	ms := NewModeSwitcher()
	ms.Switch(1000, 0) // enter stable mode

	// Collect first 20 results
	var results []bool
	for i := 0; i < 20; i++ {
		results = append(results, ms.OnDecision())
	}

	// The Luby sequence [1,1,2,1,1,2,4,...] defines the gap between true results.
	// Verify that true results appear at the expected intervals.
	truePositions := []int{}
	for i, v := range results {
		if v {
			truePositions = append(truePositions, i)
		}
	}

	// First few true positions should follow Luby gaps:
	// gap 1: positions 0, 1
	// gap 2: positions 3
	// gap 1: position 4
	// gap 1: position 5
	// gap 2: position 7
	// gap 4: position 11
	expectedPos := []int{0, 1, 3, 4, 5, 7, 11, 12, 13, 15}
	if len(truePositions) < len(expectedPos) {
		t.Fatalf("got %d true results, want at least %d: %v", len(truePositions), len(expectedPos), truePositions)
	}
	for i, want := range expectedPos {
		if truePositions[i] != want {
			t.Errorf("true position %d = %d, want %d", i, truePositions[i], want)
		}
	}
}

func TestReluctantDoublingDisabledInFocused(t *testing.T) {
	ms := NewModeSwitcher()
	// In focused mode, OnDecision should always return false
	for i := 0; i < 100; i++ {
		if ms.OnDecision() {
			t.Errorf("focused mode: OnDecision() = true at step %d, want false", i)
		}
	}
}

func TestModeSwitcherReset(t *testing.T) {
	ms := NewModeSwitcher()
	ms.Switch(1000, 0)       // focused → stable
	ms.Switch(1000, 500)     // stable → focused
	ms.Switch(2500, 500)     // focused → stable

	ms.Reset()
	if ms.Mode() != ModeFocused {
		t.Error("after Reset, should be focused")
	}
	if ms.switches != 0 {
		t.Errorf("switches = %d, want 0", ms.switches)
	}
	if ms.conflictLimit != 1000 {
		t.Errorf("conflictLimit = %d, want 1000", ms.conflictLimit)
	}
}

func TestModeSwitcherOnConflict(t *testing.T) {
	ms := NewModeSwitcher()
	// Below threshold
	if ms.OnConflict(500, 0) {
		t.Error("OnConflict should return false below threshold")
	}
	// At threshold
	if !ms.OnConflict(1000, 0) {
		t.Error("OnConflict should return true at threshold")
	}
}

func TestReluctantSequenceExtension(t *testing.T) {
	rd := &ReluctantDoubling{
		sequence: newLubySeq(),
		limit:    1,
		enabled:  true,
	}
	// Walk through the entire initial sequence
	for i := 0; i < len(rd.sequence); i++ {
		rd.index = i
		rd.limit = rd.sequence[i]
		rd.count = rd.limit // trigger
	}
	// This should extend
	oldLen := len(rd.sequence)
	rd.index = oldLen - 1
	rd.limit = rd.sequence[oldLen-1]
	rd.count = rd.limit
	// The next OnDecision call would extend via extend()
	// Simulate: set index past end to force extend
	rd.index = len(rd.sequence) // past end
	rd.extend()
	if len(rd.sequence) <= oldLen {
		t.Error("sequence should be longer after extension")
	}
}

func TestModeSwitcherMultipleCycles(t *testing.T) {
	ms := NewModeSwitcher()
	modes := []SolverMode{}
	for i := 0; i < 10; i++ {
		modes = append(modes, ms.Mode())
		if ms.Mode() == ModeFocused {
			ms.Switch(ms.conflictsAtMode+ms.conflictLimit, ms.decisionsAtMode)
		} else {
			ms.Switch(ms.conflictsAtMode, ms.decisionsAtMode+ms.tickLimit)
		}
	}
	// Should alternate
	for i := 1; i < len(modes); i++ {
		if modes[i] == modes[i-1] {
			t.Errorf("mode should alternate at step %d, got %v then %v", i, modes[i-1], modes[i])
		}
	}
}
