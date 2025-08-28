package sat

import (
	"math"
	"sort"
)

// AdvancedVSIDSHeuristic implements VSIDS with modern enhancements
type AdvancedVSIDSHeuristic struct {
	// Activity scores
	activity  map[string]float64
	increment float64
	decay     float64

	// Polarity heuristics
	polarityScores map[string]float64
	phaseCache     map[string]bool

	// LRB (Learning Rate Based) enhancement
	lrbScores map[string]float64
	lrbDecay  float64

	// Anti-aging mechanism
	participated  map[string]int64
	conflictCount int64

	// Performance optimization
	heapSize int
	heap     []string
	heapPos  map[string]int
}

func NewAdvancedVSIDSHeuristic() *AdvancedVSIDSHeuristic {
	return &AdvancedVSIDSHeuristic{
		activity:       make(map[string]float64),
		increment:      1.0,
		decay:          0.95,
		polarityScores: make(map[string]float64),
		phaseCache:     make(map[string]bool),
		lrbScores:      make(map[string]float64),
		lrbDecay:       0.8,
		participated:   make(map[string]int64),
		conflictCount:  0,
		heap:           make([]string, 0),
		heapPos:        make(map[string]int),
	}
}

func (v *AdvancedVSIDSHeuristic) Name() string {
	return "Advanced-VSIDS-LRB"
}

func (v *AdvancedVSIDSHeuristic) ChooseVariable(unassigned []string, assignment Assignment) string {
	if len(unassigned) == 0 {
		return ""
	}

	// Initialize activities for new variables
	for _, variable := range unassigned {
		if _, exists := v.activity[variable]; !exists {
			v.activity[variable] = 0.0
			v.lrbScores[variable] = 0.0
			v.polarityScores[variable] = 0.0
		}
	}

	// Find best variable using combined scoring
	bestVar := ""
	bestScore := -1.0

	for _, variable := range unassigned {
		score := v.computeCombinedScore(variable)
		if score > bestScore {
			bestScore = score
			bestVar = variable
		}
	}

	return bestVar
}

func (v *AdvancedVSIDSHeuristic) computeCombinedScore(variable string) float64 {
	vsidsScore := v.activity[variable]
	lrbScore := v.lrbScores[variable]

	// Anti-aging factor
	agingFactor := 1.0
	if participated, exists := v.participated[variable]; exists {
		age := v.conflictCount - participated
		if age > 100 {
			agingFactor = math.Exp(-float64(age-100) / 1000.0)
		}
	}

	// Combined score with weights
	return (0.7*vsidsScore + 0.3*lrbScore) * agingFactor
}

func (v *AdvancedVSIDSHeuristic) Update(conflictClause *Clause) {
	v.conflictCount++

	// Update VSIDS activities
	for _, lit := range conflictClause.Literals {
		v.activity[lit.Variable] += v.increment
		v.participated[lit.Variable] = v.conflictCount

		// Update LRB scores
		v.lrbScores[lit.Variable] = v.lrbDecay*v.lrbScores[lit.Variable] + (1.0 - v.lrbDecay)

		// Update polarity scores
		if lit.Negated {
			v.polarityScores[lit.Variable] -= 0.1
		} else {
			v.polarityScores[lit.Variable] += 0.1
		}
	}

	// Decay activities
	v.increment /= v.decay

	// Rescale if needed
	if v.increment > 1e100 {
		v.rescaleActivities()
	}

	// Update phase cache based on current assignment
	v.updatePhaseCache(conflictClause)
}

func (v *AdvancedVSIDSHeuristic) updatePhaseCache(conflictClause *Clause) {
	for _, lit := range conflictClause.Literals {
		// Cache the opposite polarity that caused conflict
		v.phaseCache[lit.Variable] = lit.Negated
	}
}

func (v *AdvancedVSIDSHeuristic) GetPreferredPolarity(variable string) bool {
	// Use phase cache if available
	if polarity, exists := v.phaseCache[variable]; exists {
		return polarity
	}

	// Use polarity scores
	if score, exists := v.polarityScores[variable]; exists {
		return score > 0.0
	}

	// Default to positive
	return true
}

func (v *AdvancedVSIDSHeuristic) rescaleActivities() {
	for variable := range v.activity {
		v.activity[variable] *= 1e-100
		v.lrbScores[variable] *= 1e-100
	}
	v.increment *= 1e-100
}

func (v *AdvancedVSIDSHeuristic) Reset() {
	v.activity = make(map[string]float64)
	v.lrbScores = make(map[string]float64)
	v.polarityScores = make(map[string]float64)
	v.phaseCache = make(map[string]bool)
	v.participated = make(map[string]int64)
	v.increment = 1.0
	v.conflictCount = 0
}

// AdaptiveRestartStrategy implements modern restart strategies
type AdaptiveRestartStrategy struct {
	// Luby sequence parameters
	lubySequence []int
	lubyIndex    int
	baseUnit     int

	// Glucose-style restart
	glucoseWindow []int64
	windowSize    int
	windowIndex   int

	// Adaptive parameters
	fastMA    float64 // Fast moving average
	slowMA    float64 // Slow moving average
	threshold float64

	// Performance tracking
	restartCount  int64
	lastConflicts int64
}

func NewAdaptiveRestartStrategy() *AdaptiveRestartStrategy {
	return &AdaptiveRestartStrategy{
		lubySequence:  []int{1, 1, 2, 1, 1, 2, 4, 1, 1, 2, 1, 1, 2, 4, 8},
		lubyIndex:     0,
		baseUnit:      100,
		glucoseWindow: make([]int64, 50),
		windowSize:    50,
		windowIndex:   0,
		fastMA:        0.0,
		slowMA:        0.0,
		threshold:     1.4,
		restartCount:  0,
		lastConflicts: 0,
	}
}

func (a *AdaptiveRestartStrategy) Name() string {
	return "Adaptive-Glucose-Luby"
}

func (a *AdaptiveRestartStrategy) ShouldRestart(stats SolverStatistics) bool {
	currentConflicts := stats.Conflicts
	recentConflicts := currentConflicts - a.lastConflicts

	// Update moving averages
	if recentConflicts > 0 {
		a.glucoseWindow[a.windowIndex] = recentConflicts
		a.windowIndex = (a.windowIndex + 1) % a.windowSize

		// Update moving averages
		alpha := 0.1
		a.fastMA = alpha*float64(recentConflicts) + (1.0-alpha)*a.fastMA
		a.slowMA = 0.01*float64(recentConflicts) + 0.99*a.slowMA
	}

	a.lastConflicts = currentConflicts

	// Glucose-style adaptive restart
	if a.restartCount > 10 && a.fastMA > a.threshold*a.slowMA {
		return true
	}

	// Luby restart as fallback
	if a.lubyIndex < len(a.lubySequence) {
		threshold := int64(a.lubySequence[a.lubyIndex] * a.baseUnit)
		return currentConflicts >= threshold
	}

	return false
}

func (a *AdaptiveRestartStrategy) OnRestart() {
	a.restartCount++
	a.lubyIndex++

	if a.lubyIndex >= len(a.lubySequence) {
		a.extendLubySequence()
	}

	// Adapt threshold based on performance
	if a.restartCount%10 == 0 {
		avgConflicts := a.computeAverageConflicts()
		if avgConflicts > 1000 {
			a.threshold *= 1.1
		} else {
			a.threshold *= 0.95
		}
	}
}

func (a *AdaptiveRestartStrategy) computeAverageConflicts() float64 {
	sum := int64(0)
	count := 0

	for _, conflicts := range a.glucoseWindow {
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

func (a *AdaptiveRestartStrategy) extendLubySequence() {
	current := len(a.lubySequence)
	for i := 0; i < current; i++ {
		a.lubySequence = append(a.lubySequence, a.lubySequence[i])
	}
	nextPower := int(math.Pow(2, float64(current)))
	a.lubySequence = append(a.lubySequence, nextPower)
}

func (a *AdaptiveRestartStrategy) Reset() {
	a.lubyIndex = 0
	a.restartCount = 0
	a.lastConflicts = 0
	a.fastMA = 0.0
	a.slowMA = 0.0
	for i := range a.glucoseWindow {
		a.glucoseWindow[i] = 0
	}
	a.windowIndex = 0
}

// AdvancedClauseDeletion implements sophisticated clause management
type AdvancedClauseDeletion struct {
	// Activity-based deletion
	activityThreshold float64

	// LBD (Literal Block Distance) based
	lbdThreshold int
	lbdScores    map[int]int

	// Size-based deletion
	sizeThreshold int

	// Performance tracking
	deletionCount int64
	keepRatio     float64
}

func NewAdvancedClauseDeletion() *AdvancedClauseDeletion {
	return &AdvancedClauseDeletion{
		activityThreshold: 0.1,
		lbdThreshold:      4,
		lbdScores:         make(map[int]int),
		sizeThreshold:     30,
		deletionCount:     0,
		keepRatio:         0.5,
	}
}

func (a *AdvancedClauseDeletion) Name() string {
	return "Advanced-LBD-Activity"
}

func (a *AdvancedClauseDeletion) ShouldDelete(clause *Clause, stats SolverStatistics) bool {
	if !clause.Learned {
		return false // Never delete original clauses
	}

	// Keep unit clauses
	if len(clause.Literals) <= 1 {
		return false
	}

	// LBD-based deletion
	lbd := a.computeLBD(clause)
	if lbd <= a.lbdThreshold {
		return false // Keep clauses with low LBD
	}

	// Activity-based deletion
	if clause.Activity < a.activityThreshold {
		return true
	}

	// Size-based deletion
	if len(clause.Literals) > a.sizeThreshold {
		return true
	}

	return false
}

func (a *AdvancedClauseDeletion) computeLBD(clause *Clause) int {
	// Simplified LBD computation
	// Real implementation would track decision levels
	return len(clause.Literals) / 2
}

func (a *AdvancedClauseDeletion) Update(clauses []*Clause) {
	if len(clauses) == 0 {
		return
	}

	// Compute activity statistics
	activities := make([]float64, 0, len(clauses))
	for _, clause := range clauses {
		if clause.Learned {
			activities = append(activities, clause.Activity)
		}
	}

	if len(activities) > 0 {
		sort.Float64s(activities)
		median := activities[len(activities)/2]
		a.activityThreshold = median * 0.3
	}

	// Adapt parameters based on performance
	a.deletionCount++
	if a.deletionCount%100 == 0 {
		// Adjust thresholds
		a.keepRatio = math.Max(0.3, a.keepRatio*0.99)
		a.sizeThreshold = int(float64(a.sizeThreshold) * 1.01)
	}
}

func (a *AdvancedClauseDeletion) Reset() {
	a.activityThreshold = 0.1
	a.deletionCount = 0
	a.keepRatio = 0.5
	a.lbdScores = make(map[int]int)
}
