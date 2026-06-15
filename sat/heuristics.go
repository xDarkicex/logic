package sat

import (
	"math"
	"sort"

	"github.com/xDarkicex/memory"
)

const (
	initVarCap      = 256  // initial variable capacity
	phaseUnset      = -1   // phaseCache sentinel: no cached phase
	phaseFalse      = 0    // phaseCache: cached false
	phaseTrue       = 1    // phaseCache: cached true
	participateUnset = -1  // participated sentinel: never seen in conflict
)

// VSIDSHeuristic implements VSIDS with modern enhancements (LRB, polarity, anti-aging)
// and a binary max-heap for O(log n) variable selection.
// All score and tracking arrays are Pool-backed; only the name→index map uses make().
type VSIDSHeuristic struct {
	// Core VSIDS — Pool-backed arrays indexed by variable handle
	activity  []float64
	increment float64
	decay     float64

	// Integrated LRB (Learning Rate Based)
	lrbScores []float64
	lrbDecay  float64

	// Integrated Polarity Heuristics
	polarity []float64
	phases   []int8 // -1=unset, 0=false, 1=true

	// Integrated Anti-aging
	participated  []int64 // -1 = never participated
	conflictCount int64

	// Configuration weights
	vsidsWeight float64
	lrbWeight   float64

	// Binary max-heap for O(log n) variable selection
	heap *VarHeap

	// Name ↔ index mapping (varIndex is the sole map — O(1) name lookup)
	varIndex map[string]int
	varNames []string
	nextVar  int
	cap      int // current capacity of all arrays
}

// NewVSIDSHeuristic creates a VSIDS heuristic with LRB, polarity, anti-aging,
// and a binary max-heap. All backing arrays are off-heap via Pool.
func NewVSIDSHeuristic() *VSIDSHeuristic {
	v := &VSIDSHeuristic{
		increment:    1.0,
		decay:        0.95,
		lrbDecay:     0.8,
		conflictCount: 0,
		vsidsWeight:  0.7,
		lrbWeight:    0.3,
		varIndex:     make(map[string]int),
		cap:          initVarCap,
	}
	v.allocArrays(initVarCap)
	v.heap = NewVarHeap(initVarCap, satPool)
	return v
}

// allocArrays allocates or re-allocates all Pool-backed arrays to the given capacity.
func (v *VSIDSHeuristic) allocArrays(cap int) {
	v.activity = memory.MustPoolSlice[float64](satPool, cap)[:cap]
	v.lrbScores = memory.MustPoolSlice[float64](satPool, cap)[:cap]
	v.polarity = memory.MustPoolSlice[float64](satPool, cap)[:cap]
	v.phases = memory.MustPoolSlice[int8](satPool, cap)[:cap]
	v.participated = memory.MustPoolSlice[int64](satPool, cap)[:cap]
	v.varNames = memory.MustPoolSlice[string](satPool, cap)[:cap]
	for i := range v.phases {
		v.phases[i] = phaseUnset
	}
	for i := range v.participated {
		v.participated[i] = participateUnset
	}
}

// grow doubles capacity and copies existing data.
func (v *VSIDSHeuristic) grow(minCap int) {
	newCap := v.cap * 2
	if newCap < minCap {
		newCap = minCap
	}

	newActivity := memory.MustPoolSlice[float64](satPool, newCap)[:newCap]
	copy(newActivity, v.activity)
	v.activity = newActivity

	newLRB := memory.MustPoolSlice[float64](satPool, newCap)[:newCap]
	copy(newLRB, v.lrbScores)
	v.lrbScores = newLRB

	newPolarity := memory.MustPoolSlice[float64](satPool, newCap)[:newCap]
	copy(newPolarity, v.polarity)
	v.polarity = newPolarity

	newPhases := memory.MustPoolSlice[int8](satPool, newCap)[:newCap]
	copy(newPhases, v.phases)
	for i := v.cap; i < newCap; i++ {
		newPhases[i] = phaseUnset
	}
	v.phases = newPhases

	newPart := memory.MustPoolSlice[int64](satPool, newCap)[:newCap]
	copy(newPart, v.participated)
	for i := v.cap; i < newCap; i++ {
		newPart[i] = participateUnset
	}
	v.participated = newPart

	newNames := memory.MustPoolSlice[string](satPool, newCap)[:newCap]
	copy(newNames, v.varNames)
	v.varNames = newNames

	v.cap = newCap
}

// Name returns the heuristic identifier.
func (v *VSIDSHeuristic) Name() string {
	return "VSIDS-LRB-Enhanced"
}

// ChooseVariable selects the next decision variable using the binary max-heap.
// Returns the unassigned variable with the highest combined activity score.
// Time: amortized O(log n). Space: O(1) additional.
func (v *VSIDSHeuristic) ChooseVariable(unassigned []string, assignment Assignment) string {
	if len(unassigned) == 0 {
		return ""
	}

	for _, name := range unassigned {
		idx := v.ensureVar(name)
		if !v.heap.Contains(idx) {
			score := v.computeScore(idx)
			v.heap.Update(idx, score)
		}
	}

	for !v.heap.IsEmpty() {
		idx := v.heap.Max()
		name := v.varNames[idx]
		if !assignment.IsAssigned(name) {
			return name
		}
		v.heap.PopMax()
	}

	return ""
}

// computeScore returns the combined VSIDS + LRB + anti-aging score for idx.
// CC=3.
func (v *VSIDSHeuristic) computeScore(idx int) float64 {
	vsidsScore := v.activity[idx]
	lrbScore := v.lrbScores[idx]

	agingFactor := 1.0
	if v.participated[idx] != participateUnset {
		age := v.conflictCount - v.participated[idx]
		if age > 100 {
			agingFactor = math.Exp(-float64(age-100) / 1000.0)
		}
	}
	return (v.vsidsWeight*vsidsScore + v.lrbWeight*lrbScore) * agingFactor
}

// ensureVar registers a variable name with the next available index.
// Returns the existing index if already registered. CC=3.
func (v *VSIDSHeuristic) ensureVar(name string) int {
	if idx, exists := v.varIndex[name]; exists {
		return idx
	}
	idx := v.nextVar
	v.nextVar++
	v.varIndex[name] = idx

	if idx >= v.cap {
		v.grow(idx + 64)
	}
	v.varNames[idx] = name
	// Arrays are already zeroed / sentinel-initialized from grow
	return idx
}

// OnBacktrack re-inserts unassigned variables that were popped from the heap
// during earlier decisions. Call after trail backtracking. CC=2.
func (v *VSIDSHeuristic) OnBacktrack(unassigned []string) {
	for _, name := range unassigned {
		idx := v.varIndex[name]
		if !v.heap.Contains(idx) {
			score := v.computeScore(idx)
			v.heap.Update(idx, score)
		}
	}
}

// Update bumps activity scores for all variables in the conflict clause
// and refreshes their positions in the decision heap. CC=4.
func (v *VSIDSHeuristic) Update(conflictClause *Clause) {
	v.conflictCount++

	for _, lit := range conflictClause.Literals {
		idx := v.ensureVar(lit.Variable)

		v.activity[idx] += v.increment
		v.lrbScores[idx] = v.lrbDecay*v.lrbScores[idx] + (1.0 - v.lrbDecay)

		if lit.Negated {
			v.polarity[idx] -= 0.1
		} else {
			v.polarity[idx] += 0.1
		}

		if lit.Negated {
			v.phases[idx] = phaseTrue
		} else {
			v.phases[idx] = phaseFalse
		}

		v.participated[idx] = v.conflictCount

		score := v.computeScore(idx)
		v.heap.Update(idx, score)
	}

	v.decayVariableActivities()

	if v.increment > 1e100 {
		v.rescaleActivities()
	}
}

func (v *VSIDSHeuristic) decayVariableActivities() {
	if v.conflictCount%1000 == 0 && v.conflictCount > 0 {
		v.adaptDecayRate()
	}
	v.increment /= v.decay
}

func (v *VSIDSHeuristic) adaptDecayRate() {
	avgActivity := v.computeAverageActivity()

	if avgActivity < 0.1 {
		v.decay *= 0.95
		if v.decay < 0.8 {
			v.decay = 0.8
		}
	} else if avgActivity > 10.0 {
		v.decay *= 1.05
		if v.decay > 0.99 {
			v.decay = 0.99
		}
	}
}

// computeAverageActivity returns the mean activity across all registered variables.
func (v *VSIDSHeuristic) computeAverageActivity() float64 {
	n := v.nextVar
	if n == 0 {
		return 0.0
	}
	sum := 0.0
	for i := 0; i < n; i++ {
		sum += v.activity[i]
	}
	return sum / float64(n)
}

// GetPreferredPolarity returns the cached or score-derived preferred polarity for variable.
func (v *VSIDSHeuristic) GetPreferredPolarity(variable string) bool {
	idx, ok := v.varIndex[variable]
	if !ok {
		return true
	}
	if v.phases[idx] != phaseUnset {
		return v.phases[idx] == phaseTrue
	}
	return v.polarity[idx] > 0.0
}

func (v *VSIDSHeuristic) rescaleActivities() {
	factor := 1e-100
	n := v.nextVar
	for i := 0; i < n; i++ {
		v.activity[i] *= factor
		v.lrbScores[i] *= factor
	}
	v.increment *= factor
	v.heap.Rescale(factor)
}

// Reset clears all heuristic state for solver reuse.
func (v *VSIDSHeuristic) Reset() {
	v.heap.Reset()
	v.allocArrays(initVarCap)
	v.varIndex = make(map[string]int)
	v.nextVar = 0
	v.cap = initVarCap
	v.increment = 1.0
	v.decay = 0.95
	v.lrbDecay = 0.8
	v.conflictCount = 0
}

// RandomHeuristic chooses variables randomly (for comparison).
type RandomHeuristic struct{}

func NewRandomHeuristic() *RandomHeuristic { return &RandomHeuristic{} }

func (r *RandomHeuristic) Name() string { return "Random" }

func (r *RandomHeuristic) ChooseVariable(unassigned []string, assignment Assignment) string {
	if len(unassigned) == 0 {
		return ""
	}
	return unassigned[0]
}

func (r *RandomHeuristic) Update(conflictClause *Clause)       {}
func (r *RandomHeuristic) OnBacktrack(unassigned []string)      {}
func (r *RandomHeuristic) Reset()                               {}

// LubyRestartStrategy implements hybrid Luby + Glucose-style adaptive restarts.
type LubyRestartStrategy struct {
	sequence []int
	index    int
	baseUnit int

	glucoseWindow []int64
	windowSize    int
	windowIndex   int
	fastMA        float64
	slowMA        float64
	threshold     float64

	restartCount  int64
	lastConflicts int64
}

// NewLubyRestartStrategy creates a hybrid Luby + adaptive Glucose restart strategy.
func NewLubyRestartStrategy() *LubyRestartStrategy {
	return &LubyRestartStrategy{
		sequence:      []int{1, 1, 2, 1, 1, 2, 4, 1, 1, 2, 1, 1, 2, 4, 8},
		index:         0,
		baseUnit:      100,
		glucoseWindow: memory.MustPoolSlice[int64](satPool, 50)[:50],
		windowSize:    50,
		threshold:     1.4,
	}
}

func (l *LubyRestartStrategy) Name() string { return "Luby-Adaptive-Glucose" }

// ShouldRestart returns true if the solver should restart based on Luby or Glucose criteria.
func (l *LubyRestartStrategy) ShouldRestart(stats SolverStatistics) bool {
	currentConflicts := stats.Conflicts
	recentConflicts := currentConflicts - l.lastConflicts

	if recentConflicts > 0 {
		l.glucoseWindow[l.windowIndex] = recentConflicts
		l.windowIndex = (l.windowIndex + 1) % l.windowSize

		alpha := 0.1
		l.fastMA = alpha*float64(recentConflicts) + (1.0-alpha)*l.fastMA
		l.slowMA = 0.01*float64(recentConflicts) + 0.99*l.slowMA
	}

	l.lastConflicts = currentConflicts

	if l.restartCount > 10 && l.fastMA > l.threshold*l.slowMA {
		return true
	}

	if l.index < len(l.sequence) {
		threshold := int64(l.sequence[l.index] * l.baseUnit)
		return currentConflicts >= threshold
	}

	return false
}

// OnRestart advances the restart sequence.
func (l *LubyRestartStrategy) OnRestart() {
	l.restartCount++
	l.index++
	if l.index >= len(l.sequence) {
		l.extendSequence()
	}

	if l.restartCount%10 == 0 {
		avgConflicts := l.computeAverageConflicts()
		if avgConflicts > 1000 {
			l.threshold *= 1.1
		} else {
			l.threshold *= 0.95
		}
	}
}

func (l *LubyRestartStrategy) computeAverageConflicts() float64 {
	sum := int64(0)
	count := 0
	for _, conflicts := range l.glucoseWindow {
		if conflicts > 0 {
			sum += conflicts
			count++
		}
	}
	if count == 0 {
		return 0.0
	}
	return float64(sum) / float64(count)
}

// Reset clears restart state.
func (l *LubyRestartStrategy) Reset() {
	l.index = 0
	l.restartCount = 0
	l.lastConflicts = 0
	l.fastMA = 0.0
	l.slowMA = 0.0
	for i := range l.glucoseWindow {
		l.glucoseWindow[i] = 0
	}
	l.windowIndex = 0
}

func (l *LubyRestartStrategy) extendSequence() {
	current := len(l.sequence)
	for i := 0; i < current; i++ {
		l.sequence = append(l.sequence, l.sequence[i])
	}
	l.sequence = append(l.sequence, int(math.Pow(2, float64(len(l.sequence)))))
}

// ActivityBasedDeletion implements LBD + tier-aware clause deletion.
type ActivityBasedDeletion struct {
	activityThreshold float64
	lbdThreshold      int
	sizeThreshold     int
	deletionCount     int64
	keepRatio         float64
	coreProtection    bool
	midThreshold      float64
	localThreshold    float64
	recentProtection  int64
}

// NewActivityBasedDeletion creates a tier-aware clause deletion policy.
func NewActivityBasedDeletion() *ActivityBasedDeletion {
	return &ActivityBasedDeletion{
		activityThreshold: 0.1,
		lbdThreshold:      4,
		sizeThreshold:     30,
		keepRatio:         0.5,
		coreProtection:    true,
		midThreshold:      0.15,
		localThreshold:    0.10,
		recentProtection:  1000,
	}
}

func (a *ActivityBasedDeletion) Name() string { return "Activity-LBD-Enhanced" }

// ShouldDelete returns true if the clause should be deleted based on tier and activity.
func (a *ActivityBasedDeletion) ShouldDelete(clause *Clause, stats SolverStatistics) bool {
	return a.ShouldDeleteFromTier(clause, clause.Tier, stats)
}

// ShouldDeleteFromTier applies tier-specific deletion criteria.
func (a *ActivityBasedDeletion) ShouldDeleteFromTier(clause *Clause, tier int, stats SolverStatistics) bool {
	if !clause.Learned || len(clause.Literals) <= 1 {
		return false
	}
	if (a.coreProtection && tier == 0) || clause.Glue || clause.LBD <= 2 || tier == 0 {
		return false
	}
	if tier == 1 {
		return clause.Activity < a.midThreshold
	}
	if tier == 2 {
		if clause.Activity < a.localThreshold || len(clause.Literals) > a.sizeThreshold {
			return true
		}
		return clause.Activity < a.activityThreshold
	}
	return clause.Activity < a.activityThreshold
}

// GetDeletionCandidates selects clauses for deletion, preferring local tier then mid tier.
func (a *ActivityBasedDeletion) GetDeletionCandidates(db *ClauseDatabase, stats SolverStatistics) []*Clause {
	need := db.Size() - db.maxSize
	if need <= 0 {
		return nil
	}

	var out []*Clause

	pick := func(tierClauses []*Clause, tier int) {
		for _, cl := range tierClauses {
			if need == 0 {
				return
			}
			if a.ShouldDeleteFromTier(cl, tier, stats) {
				out = append(out, cl)
				need--
			}
		}
	}

	_, mid, local, recent := db.GetTierSlices()
	_ = recent

	pick(local, 2)
	if need > 0 {
		pick(mid, 1)
	}

	return out
}

// Update adjusts thresholds based on observed LBD and activity distributions.
func (a *ActivityBasedDeletion) Update(clauses []*Clause) {
	if len(clauses) == 0 {
		return
	}

	var lbdSum int
	var clauseCount int
	lbdCounts := make(map[int]int)
	activities := memory.MustPoolSlice[float64](satPool, len(clauses))

	for _, clause := range clauses {
		if clause.Learned {
			lbdCounts[clause.LBD]++
			lbdSum += clause.LBD
			clauseCount++
			activities = append(activities, clause.Activity)
		}
	}

	if len(activities) > 0 {
		sort.Float64s(activities)
		median := activities[len(activities)/2]
		a.activityThreshold = median * 0.3

		avgLBD := float64(lbdSum) / float64(clauseCount)
		if avgLBD < 4.0 {
			a.lbdThreshold = 3
		} else {
			a.lbdThreshold = 4
		}
	}

	a.deletionCount++
	if a.deletionCount%100 == 0 {
		glueSum := 0
		if val, ok := lbdCounts[1]; ok {
			glueSum += val
		}
		if val, ok := lbdCounts[2]; ok {
			glueSum += val
		}

		glueRatio := float64(glueSum) / float64(max(1, clauseCount))
		if glueRatio > 0.3 {
			if a.keepRatio < 0.4 {
				a.keepRatio = 0.4
			} else {
				a.keepRatio *= 1.01
			}
		} else {
			if a.keepRatio < 0.3 {
				a.keepRatio = 0.3
			} else {
				a.keepRatio *= 0.99
			}
		}
	}
}

// Reset clears deletion policy state.
func (a *ActivityBasedDeletion) Reset() {
	a.activityThreshold = 0.1
	a.deletionCount = 0
	a.keepRatio = 0.5
	a.lbdThreshold = 4
}

// NewAdvancedVSIDSHeuristic returns an enhanced VSIDS heuristic (alias for NewVSIDSHeuristic).
func NewAdvancedVSIDSHeuristic() *VSIDSHeuristic { return NewVSIDSHeuristic() }

// NewAdaptiveRestartStrategy returns an enhanced restart strategy (alias for NewLubyRestartStrategy).
func NewAdaptiveRestartStrategy() *LubyRestartStrategy { return NewLubyRestartStrategy() }

// NewAdvancedClauseDeletion returns an enhanced deletion policy (alias for NewActivityBasedDeletion).
func NewAdvancedClauseDeletion() *ActivityBasedDeletion { return NewActivityBasedDeletion() }
