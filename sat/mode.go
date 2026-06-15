package sat

import "github.com/xDarkicex/memory"

// SolverMode represents the search strategy mode of the CDCL solver.
type SolverMode int

const (
	// ModeFocused exploits promising regions with aggressive VSIDS bumps
	// and rapid EMA decay. This is the default mode.
	ModeFocused SolverMode = iota

	// ModeStable explores new variable orderings with reluctant doubling
	// and periodic score recalculation instead of per-conflict bumping.
	ModeStable
)

// ModeSwitcher manages focused/stable mode alternation.
// The reluctant doubling sequence is Pool-backed; all other state is value-type.
// CC ≤ 4 on all methods.
type ModeSwitcher struct {
	mode SolverMode

	// Base intervals (fixed); limits scale by switch count
	baseConflict int64
	baseTick     int64
	conflictLimit int64
	tickLimit     int64

	switches        int64
	conflictsAtMode int64
	decisionsAtMode int64

	reluctant ReluctantDoubling
}

// ReluctantDoubling produces a Luby-like sequence determining when to make
// a random decision during stable mode. Sequence backing is Pool-allocated.
type ReluctantDoubling struct {
	sequence []int
	index    int
	limit    int
	count    int
	enabled  bool
}

// newLubySeq creates a Pool-backed Luby sequence for reluctant doubling.
func newLubySeq() []int {
	s := memory.MustPoolSlice[int](satPool, 15)[:15]
	copy(s, []int{1, 1, 2, 1, 1, 2, 4, 1, 1, 2, 1, 1, 2, 4, 8})
	return s
}

// NewModeSwitcher creates a mode switcher starting in focused mode.
func NewModeSwitcher() *ModeSwitcher {
	return &ModeSwitcher{
		mode:          ModeFocused,
		baseConflict:  1000,
		baseTick:      500,
		conflictLimit: 1000,
		tickLimit:     500,
		reluctant: ReluctantDoubling{
			sequence: newLubySeq(),
			limit:    1,
		},
	}
}

// Mode returns the current solver mode.
func (ms *ModeSwitcher) Mode() SolverMode { return ms.mode }

// ShouldSwitch reports whether the mode should change based on current counts.
// CC=2.
func (ms *ModeSwitcher) ShouldSwitch(conflicts, decisions int64) bool {
	switch ms.mode {
	case ModeFocused:
		return conflicts-ms.conflictsAtMode >= ms.conflictLimit
	case ModeStable:
		return decisions-ms.decisionsAtMode >= ms.tickLimit
	}
	return false
}

// Switch toggles between focused and stable mode and scales the next threshold.
// CC=2.
func (ms *ModeSwitcher) Switch(conflicts, decisions int64) {
	ms.switches++
	if ms.mode == ModeFocused {
		ms.mode = ModeStable
		ms.decisionsAtMode = decisions
		ms.tickLimit = ms.scaleLimit(ms.baseTick)
		ms.reluctant.Reset()
		ms.reluctant.enabled = true
	} else {
		ms.mode = ModeFocused
		ms.conflictsAtMode = conflicts
		ms.conflictLimit = ms.scaleLimit(ms.baseConflict)
		ms.reluctant.enabled = false
	}
}

// scaleLimit scales the base interval by switch count squared.
// As the solver matures, modes run longer before switching. CC=1.
func (ms *ModeSwitcher) scaleLimit(base int64) int64 {
	n := ms.switches/2 + 1
	return base * n * n
}

// OnConflict updates counters when a conflict occurs. Call from the CDCL loop.
// Returns true if the mode should switch.
// CC=1.
func (ms *ModeSwitcher) OnConflict(conflicts, decisions int64) bool {
	return ms.ShouldSwitch(conflicts, decisions)
}

// OnDecision should be called each time the solver makes a decision.
// Returns true if this decision should be random (reluctant doubling trigger).
// CC=2.
func (ms *ModeSwitcher) OnDecision() bool {
	if !ms.reluctant.enabled {
		return false
	}
	ms.reluctant.count++
	if ms.reluctant.count >= ms.reluctant.limit {
		ms.reluctant.count = 0
		ms.reluctant.index++
		if ms.reluctant.index >= len(ms.reluctant.sequence) {
			ms.reluctant.extend()
		}
		ms.reluctant.limit = ms.reluctant.sequence[ms.reluctant.index]
		return true
	}
	return false
}

// Reset clears all state for solver reuse.
func (ms *ModeSwitcher) Reset() {
	ms.mode = ModeFocused
	ms.baseConflict = 1000
	ms.baseTick = 500
	ms.conflictLimit = 1000
	ms.tickLimit = 500
	ms.switches = 0
	ms.conflictsAtMode = 0
	ms.decisionsAtMode = 0
	ms.reluctant.sequence = newLubySeq()
	ms.reluctant.limit = 1
	ms.reluctant.Reset()
}

// Reset resets the reluctant doubling counters. CC=1.
func (rd *ReluctantDoubling) Reset() {
	rd.index = 0
	rd.limit = rd.sequence[0]
	rd.count = 0
	rd.enabled = false
}

// extend grows the Pool-backed sequence by repeating values and appending
// a power-of-two. Mirrors Kissat's Luby extension. CC=2.
func (rd *ReluctantDoubling) extend() {
	cur := len(rd.sequence)
	newCap := cur*2 + 1
	newSeq := memory.MustPoolSlice[int](satPool, newCap)[:newCap]
	copy(newSeq, rd.sequence)
	copy(newSeq[cur:], rd.sequence)
	newSeq[newCap-1] = 1 << (newCap / cur)
	rd.sequence = newSeq
}
