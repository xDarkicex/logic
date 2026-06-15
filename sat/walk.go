package sat

import "github.com/xDarkicex/memory"

const (
	walkMaxScoreTable = 20
	walkDefaultCB     = 2.0
	walkDefaultFlips  = 10000
)

// walkCounter tracks per-clause satisfaction state during WalkSAT.
type walkCounter struct {
	count uint32
	pos   int32
}

// WalkSolver performs WalkSAT-style local search on irredundant clauses
// to find an initial assignment or warm-start phases for CDCL.
// All backing arrays are Pool-allocated. Not safe for concurrent use.
type WalkSolver struct {
	varIndex map[string]int
	varNames []string
	numVars  int

	clauses  []*Clause
	counters []walkCounter
	unsat    []int
	unsatSz  int

	occurrences [][]int

	values     []int8
	bestValues []int8
	bestUnsat  int

	scoreTable []float64
	scores     []float64

	rng uint64

	maxFlips int64
	flips    int64
}

// NewWalkSolver creates a WalkSAT solver with default parameters.
// Backing arrays are allocated lazily from satPool on the first Solve call.
func NewWalkSolver() *WalkSolver {
	return &WalkSolver{
		varIndex: make(map[string]int),
		maxFlips: walkDefaultFlips,
		rng:      0x9e3779b97f4a7c15,
	}
}

// Solve runs WalkSAT local search on irredundant clauses.
// Returns true if a satisfying assignment was found.
// Best phases are available via ExportPhases regardless of result.
// CC=4.
func (w *WalkSolver) Solve(clauses []*Clause) bool {
	if len(clauses) == 0 {
		return true
	}
	w.initFromClauses(clauses)
	w.buildOccurrences(clauses)
	w.initCounters(clauses)

	w.bestUnsat = w.unsatSz
	w.saveBest()
	if w.unsatSz == 0 {
		return true
	}

	for w.flips < w.maxFlips {
		if w.unsatSz == 0 {
			return true
		}
		w.walkStep()
	}
	return false
}

// initFromClauses builds the variable name→index mapping.
// Uses varIndex as a temporary set then rebuilds with proper indices.
// CC=3.
func (w *WalkSolver) initFromClauses(clauses []*Clause) {
	for _, c := range clauses {
		for _, lit := range c.Literals {
			if _, exists := w.varIndex[lit.Variable]; !exists {
				w.varIndex[lit.Variable] = -1
			}
		}
	}

	w.numVars = len(w.varIndex)
	w.varNames = memory.MustPoolSlice[string](satPool, w.numVars)[:w.numVars]

	for k := range w.varIndex {
		delete(w.varIndex, k)
	}

	for _, c := range clauses {
		for _, lit := range c.Literals {
			if _, exists := w.varIndex[lit.Variable]; !exists {
				idx := len(w.varIndex)
				w.varIndex[lit.Variable] = idx
				w.varNames[idx] = lit.Variable
			}
		}
	}

	n := w.numVars
	w.values = memory.MustPoolSlice[int8](satPool, n)[:n]
	w.bestValues = memory.MustPoolSlice[int8](satPool, n)[:n]
	for i := range w.values {
		w.values[i] = -1
		w.bestValues[i] = -1
	}

	w.scoreTable = memory.MustPoolSlice[float64](satPool, walkMaxScoreTable)[:walkMaxScoreTable]
	w.buildScoreTable()
}

// buildScoreTable pre-computes exponential weights: table[b] = 1/CB^b.
// CC=1.
func (w *WalkSolver) buildScoreTable() {
	v := 1.0
	for i := range w.scoreTable {
		w.scoreTable[i] = v
		v /= walkDefaultCB
	}
}

// buildOccurrences constructs literal→clause occurrence lists from Pool.
// CC=3.
func (w *WalkSolver) buildOccurrences(clauses []*Clause) {
	numLits := 2 * w.numVars
	counts := memory.MustPoolSlice[int](satPool, numLits)[:numLits]

	for ci, c := range clauses {
		for _, lit := range c.Literals {
			counts[w.litIdx(lit.Variable, lit.Negated)]++
		}
		_ = ci
	}

	w.occurrences = memory.MustPoolSlice[[]int](satPool, numLits)[:numLits]
	for i := range w.occurrences {
		if counts[i] > 0 {
			w.occurrences[i] = memory.MustPoolSlice[int](satPool, counts[i])[:0]
		}
	}

	for ci, c := range clauses {
		for _, lit := range c.Literals {
			li := w.litIdx(lit.Variable, lit.Negated)
			w.occurrences[li] = append(w.occurrences[li], ci)
		}
		_ = ci
	}
}

// initCounters sets initial clause satisfaction state.
// Default assignment: all variables true. Populates unsat stack.
// CC=3.
func (w *WalkSolver) initCounters(clauses []*Clause) {
	nc := len(clauses)
	w.clauses = clauses
	w.counters = memory.MustPoolSlice[walkCounter](satPool, nc)[:nc]
	w.unsat = memory.MustPoolSlice[int](satPool, nc)[:0]
	w.scores = memory.MustPoolSlice[float64](satPool, 32)[:0]

	for _, c := range clauses {
		for _, lit := range c.Literals {
			idx := w.varIndex[lit.Variable]
			if w.values[idx] == -1 {
				w.values[idx] = 1
			}
		}
	}

	for ci, c := range clauses {
		var sat uint32
		for _, lit := range c.Literals {
			idx := w.varIndex[lit.Variable]
			val := w.values[idx]
			if val == -1 {
				val = 1
			}
			if (!lit.Negated && val > 0) || (lit.Negated && val == 0) {
				sat++
			}
		}
		w.counters[ci].count = sat
		if sat == 0 {
			w.counters[ci].pos = int32(len(w.unsat))
			w.unsat = append(w.unsat, ci)
		} else {
			w.counters[ci].pos = -1
		}
	}
	w.unsatSz = len(w.unsat)
}

// walkStep performs one flip iteration: pick random unsat clause,
// pick best literal by break score, flip, update counters. CC≤7.
func (w *WalkSolver) walkStep() {
	ci := w.unsat[w.randIntn(w.unsatSz)]
	clause := w.clauses[ci]

	if cap(w.scores) < len(clause.Literals) {
		w.scores = memory.MustPoolSlice[float64](satPool, len(clause.Literals))[:0]
	}
	w.scores = w.scores[:0]

	var sum float64
	for _, lit := range clause.Literals {
		idx := w.varIndex[lit.Variable]
		breaks := w.breakCount(idx, lit.Negated)
		score := w.lookupScore(breaks)
		w.scores = append(w.scores, score)
		sum += score
	}

	if sum <= 0 {
		return
	}

	threshold := w.randFloat() * sum
	var accum float64
	pick := 0
	for i, s := range w.scores {
		accum += s
		if threshold < accum {
			pick = i
			break
		}
	}

	w.flipLiteral(clause.Literals[pick])
	w.flips++

	if w.unsatSz < w.bestUnsat {
		w.saveBest()
	}
}

// breakCount returns how many clauses would become unsatisfied if lit is flipped.
// CC=2.
func (w *WalkSolver) breakCount(varIdx int, negated bool) int {
	opp := w.litIdx2(varIdx, !negated)
	count := 0
	for _, ci := range w.occurrences[opp] {
		if w.counters[ci].count == 1 {
			count++
		}
	}
	return count
}

// lookupScore returns the exponential weight for a break count. CC=1.
func (w *WalkSolver) lookupScore(breaks int) float64 {
	if breaks < len(w.scoreTable) {
		return w.scoreTable[breaks]
	}
	return w.scoreTable[len(w.scoreTable)-1]
}

// flipLiteral toggles a literal and updates all clause counters. CC=5.
func (w *WalkSolver) flipLiteral(lit Literal) {
	idx := w.varIndex[lit.Variable]

	// Flipping a literal makes it true:
	//   flip("A")  → A=true (1);  flip("¬A") → ¬A=true → A=false (0)
	var newVal int8
	if lit.Negated {
		newVal = 0
	} else {
		newVal = 1
	}

	w.values[idx] = newVal

	// Clauses containing this literal (now satisfied): increment
	posLi := w.litIdx2(idx, lit.Negated)
	for _, ci := range w.occurrences[posLi] {
		if w.counters[ci].count == 0 {
			w.removeUnsat(ci)
		}
		w.counters[ci].count++
	}

	// Clauses containing negation (now less satisfied): decrement
	negLi := w.litIdx2(idx, !lit.Negated)
	for _, ci := range w.occurrences[negLi] {
		if w.counters[ci].count == 1 {
			w.addUnsat(ci)
		}
		if w.counters[ci].count > 0 {
			w.counters[ci].count--
		}
	}
}

// removeUnsat removes clause ci from the unsat stack. CC=2.
func (w *WalkSolver) removeUnsat(ci int) {
	pos := w.counters[ci].pos
	if pos < 0 {
		return
	}
	last := w.unsat[w.unsatSz-1]
	w.unsat[pos] = last
	w.counters[last].pos = pos
	w.counters[ci].pos = -1
	w.unsatSz--
}

// addUnsat appends clause ci to the unsat stack. CC=1.
func (w *WalkSolver) addUnsat(ci int) {
	if w.unsatSz >= len(w.unsat) {
		w.unsat = append(w.unsat, ci)
	} else {
		w.unsat[w.unsatSz] = ci
	}
	w.counters[ci].pos = int32(w.unsatSz)
	w.unsatSz++
}

// saveBest copies the current assignment as the best known. CC=1.
func (w *WalkSolver) saveBest() {
	copy(w.bestValues, w.values)
	w.bestUnsat = w.unsatSz
}

// ExportPhases copies the best WalkSAT assignment into the VSIDS polarity cache
// to warm-start CDCL with promising initial decision phases. CC=2.
func (w *WalkSolver) ExportPhases(heuristic *VSIDSHeuristic) {
	for i, val := range w.bestValues {
		if val == -1 || i >= len(w.varNames) {
			continue
		}
		name := w.varNames[i]
		if name == "" {
			continue
		}
		idx := heuristic.ensureVar(name)
		if val > 0 {
			heuristic.phases[idx] = phaseTrue
		} else {
			heuristic.phases[idx] = phaseFalse
		}
	}
}

// Reset clears all state for reuse. CC=1.
func (w *WalkSolver) Reset() {
	w.varIndex = make(map[string]int)
	w.varNames = nil
	w.numVars = 0
	w.clauses = nil
	w.counters = nil
	w.unsat = nil
	w.unsatSz = 0
	w.occurrences = nil
	w.values = nil
	w.bestValues = nil
	w.bestUnsat = 0
	w.scoreTable = nil
	w.scores = nil
	w.flips = 0
	w.rng = 0x9e3779b97f4a7c15
}

func (w *WalkSolver) litIdx(variable string, negated bool) int {
	return w.litIdx2(w.varIndex[variable], negated)
}

func (w *WalkSolver) litIdx2(varIdx int, negated bool) int {
	if negated {
		return 2*varIdx + 1
	}
	return 2 * varIdx
}

func (w *WalkSolver) randIntn(n int) int {
	w.rng ^= w.rng << 13
	w.rng ^= w.rng >> 7
	w.rng ^= w.rng << 17
	return int(w.rng % uint64(n))
}

func (w *WalkSolver) randFloat() float64 {
	w.rng ^= w.rng << 13
	w.rng ^= w.rng >> 7
	w.rng ^= w.rng << 17
	return float64(w.rng>>11) / float64(1<<53)
}

// UnsatCount returns the number of currently unsatisfied clauses.
func (w *WalkSolver) UnsatCount() int { return w.unsatSz }

// FlipCount returns the number of flips performed.
func (w *WalkSolver) FlipCount() int64 { return w.flips }
