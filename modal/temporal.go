package modal

import (
	"github.com/xDarkicex/memory"
)

// TemporalModel extends a Kripke model with an ordered timeline of worlds.
// Each position in the timeline represents a discrete time step (session turn).
// Evaluation of temporal operators uses the timeline order: future = later positions.
type TemporalModel struct {
	*Model
	timeline []World // Arena-backed, sequential [0, 1, 2, ...]
}

// NewTemporalModel creates a TemporalModel with the given timeline length.
// The timeline is pre-populated with sequentially numbered worlds.
func NewTemporalModel(frame *Frame, numWorlds, numVars int, pool *memory.Pool, arena *memory.Arena) *TemporalModel {
	tl := memory.MustArenaSlice[World](arena, numWorlds)
	tl = tl[:numWorlds]
	for i := 0; i < numWorlds; i++ {
		tl[i] = World(i)
	}
	return &TemporalModel{
		Model:    NewModel(frame, numVars, pool, arena),
		timeline: tl,
	}
}

// Len returns the number of time steps in the timeline.
func (tm *TemporalModel) Len() int { return len(tm.timeline) }

// Timeline returns the ordered world sequence.
func (tm *TemporalModel) Timeline() []World { return tm.timeline }

// EvalAlways evaluates □p at world w: p must hold at w and all future worlds.
// Returns the minimum truth value across w and all later positions.
// O(T) where T = timeline length from w onward. CC=3.
func (tm *TemporalModel) EvalAlways(p Formula, w World) (TruthValue, error) {
	start := int(w)
	if start >= len(tm.timeline) {
		return 1.0, nil
	}
	result := TruthValue(1.0)
	for i := start; i < len(tm.timeline); i++ {
		tv, err := p.Evaluate(tm.timeline[i], tm.Model)
		if err != nil {
			return 0, err
		}
		if tv < result {
			result = tv
		}
	}
	return result, nil
}

// EvalEventually evaluates ◇p at world w: p must hold at w or some future world.
// Returns the maximum truth value across w and all later positions.
// O(T). CC=3.
func (tm *TemporalModel) EvalEventually(p Formula, w World) (TruthValue, error) {
	start := int(w)
	if start >= len(tm.timeline) {
		return 0.0, nil
	}
	result := TruthValue(0.0)
	for i := start; i < len(tm.timeline); i++ {
		tv, err := p.Evaluate(tm.timeline[i], tm.Model)
		if err != nil {
			return 0, err
		}
		if tv > result {
			result = tv
		}
	}
	return result, nil
}

// EvalNext evaluates ○p at world w: p must hold at the next world.
// Returns false (0.0) if w is the last world (no successor).
// O(1). CC=2.
func (tm *TemporalModel) EvalNext(p Formula, w World) (TruthValue, error) {
	next := int(w) + 1
	if next >= len(tm.timeline) {
		return 0.0, nil
	}
	return p.Evaluate(tm.timeline[next], tm.Model)
}

// EvalUntil evaluates p U q at world w: p must hold at every world from w
// until a world where q holds. Returns 1.0 if q holds at or after w with p
// holding at all intermediate worlds. Returns q's truth at the first satisfaction.
// O(T²) worst case. CC=5.
func (tm *TemporalModel) EvalUntil(p, q Formula, w World) (TruthValue, error) {
	start := int(w)
	n := len(tm.timeline)
	if start >= n {
		return 0.0, nil
	}
	// For each possible position where q might hold
	var best TruthValue
	for k := start; k < n; k++ {
		qtv, err := q.Evaluate(tm.timeline[k], tm.Model)
		if err != nil {
			return 0, err
		}
		if qtv == 0 {
			continue
		}
		// Check p holds at all positions from start to k-1
		holds := true
		for j := start; j < k; j++ {
			ptv, err := p.Evaluate(tm.timeline[j], tm.Model)
			if err != nil {
				return 0, err
			}
			if ptv == 0 {
				holds = false
				break
			}
		}
		if holds {
			if qtv > best {
				best = qtv
			}
		}
	}
	return best, nil
}

// EvalWeakUntil evaluates p W q at world w: like Until but q may never hold.
// If q never holds, p must hold forever. Returns 1.0 if satisfied.
// O(T). CC=4.
func (tm *TemporalModel) EvalWeakUntil(p, q Formula, w World) (TruthValue, error) {
	start := int(w)
	n := len(tm.timeline)
	if start >= n {
		return 0.0, nil
	}
	// Check if q holds somewhere, with p holding before it
	for k := start; k < n; k++ {
		qtv, err := q.Evaluate(tm.timeline[k], tm.Model)
		if err != nil {
			return 0, err
		}
		if qtv > 0 {
			// Check p holds from start to k-1
			for j := start; j < k; j++ {
				ptv, err := p.Evaluate(tm.timeline[j], tm.Model)
				if err != nil {
					return 0, err
				}
				if ptv == 0 {
					return 0.0, nil
				}
			}
			return qtv, nil
		}
	}
	// q never holds — p must hold forever
	for i := start; i < n; i++ {
		ptv, err := p.Evaluate(tm.timeline[i], tm.Model)
		if err != nil {
			return 0, err
		}
		if ptv == 0 {
			return 0.0, nil
		}
	}
	return 1.0, nil
}
