package sat

import (
	"math"
	"sort"
)

// VSIDSHeuristic implements VSIDS with modern enhancements (LRB, polarity, anti-aging)
type VSIDSHeuristic struct {
	// Core VSIDS (existing)
	activity  map[string]float64
	increment float64
	decay     float64

	// Integrated LRB (Learning Rate Based)
	lrbScores map[string]float64
	lrbDecay  float64

	// Integrated Polarity Heuristics
	polarityScores map[string]float64
	phaseCache     map[string]bool

	// Integrated Anti-aging
	participated  map[string]int64
	conflictCount int64

	// Configuration weights
	vsidsWeight float64
	lrbWeight   float64
}

// NewVSIDSHeuristic - now returns modern implementation with all features
func NewVSIDSHeuristic() *VSIDSHeuristic {
	return &VSIDSHeuristic{
		// Core VSIDS
		activity:  make(map[string]float64),
		increment: 1.0,
		decay:     0.95,

		// Advanced features - ALL enabled by default
		lrbScores:      make(map[string]float64),
		lrbDecay:       0.8,
		polarityScores: make(map[string]float64),
		phaseCache:     make(map[string]bool),
		participated:   make(map[string]int64),
		conflictCount:  0,

		// Modern balanced weights
		vsidsWeight: 0.7,
		lrbWeight:   0.3,
	}
}

func (v *VSIDSHeuristic) Name() string {
	return "VSIDS-LRB-Enhanced"
}

// Enhanced ChooseVariable with modern combined scoring
func (v *VSIDSHeuristic) ChooseVariable(unassigned []string, assignment Assignment) string {
	if len(unassigned) == 0 {
		return ""
	}

	// Initialize new variables
	for _, variable := range unassigned {
		if _, exists := v.activity[variable]; !exists {
			v.activity[variable] = 0.0
			v.lrbScores[variable] = 0.0
			v.polarityScores[variable] = 0.0
		}
	}

	// Modern combined scoring
	bestVar := ""
	bestScore := -1.0
	for _, variable := range unassigned {
		score := v.computeModernScore(variable)
		if score > bestScore {
			bestScore = score
			bestVar = variable
		}
	}
	return bestVar // FIXED: Return bestVar instead of unassigned
}

// Modern scoring combining VSIDS + LRB + anti-aging
func (v *VSIDSHeuristic) computeModernScore(variable string) float64 {
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
	return (v.vsidsWeight*vsidsScore + v.lrbWeight*lrbScore) * agingFactor
}

// Enhanced Update with all modern techniques
func (v *VSIDSHeuristic) Update(conflictClause *Clause) {
	v.conflictCount++

	for _, lit := range conflictClause.Literals {
		// Core VSIDS update
		v.activity[lit.Variable] += v.increment

		// LRB update
		v.lrbScores[lit.Variable] = v.lrbDecay*v.lrbScores[lit.Variable] + (1.0 - v.lrbDecay)

		// Polarity update
		if lit.Negated {
			v.polarityScores[lit.Variable] -= 0.1
		} else {
			v.polarityScores[lit.Variable] += 0.1
		}

		// Phase cache update (opposite polarity that caused conflict)
		v.phaseCache[lit.Variable] = lit.Negated

		// Anti-aging tracking
		v.participated[lit.Variable] = v.conflictCount
	}

	// Adaptive VSIDS decay routed through the enhanced decay function
	v.decayVariableActivities()

	// Keep increment numerically stable
	if v.increment > 1e100 {
		v.rescaleActivities()
	}
}

// decayVariableActivities applies (possibly adaptive) VSIDS decay to the increment.
// This is designed to be called from the CDCL loop and within Update for cohesion.
func (v *VSIDSHeuristic) decayVariableActivities() {
	// Apply adaptive decay adjustment periodically
	if v.conflictCount%1000 == 0 && v.conflictCount > 0 {
		v.adaptDecayRate()
	}

	// Apply the (possibly adjusted) decay
	// Map: varActivityInc -> increment, varActivityDecay -> decay
	v.increment /= v.decay
}

// adaptDecayRate adapts v.decay based on average activity to stabilize scoring dynamics.
func (v *VSIDSHeuristic) adaptDecayRate() {
	avgActivity := v.computeAverageActivity()

	if avgActivity < 0.1 {
		// Slower decay when activities are low
		v.decay *= 0.95
		// Prevent too slow decay (keeps division by decay well-conditioned)
		if v.decay < 0.8 {
			v.decay = 0.8
		}
	} else if avgActivity > 10.0 {
		// Faster decay when activities are high
		v.decay *= 1.05
		// Prevent too fast decay
		if v.decay > 0.99 {
			v.decay = 0.99
		}
	}
}

// computeAverageActivity computes the average variable activity used by adaptDecayRate.
func (v *VSIDSHeuristic) computeAverageActivity() float64 {
	if len(v.activity) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, a := range v.activity {
		sum += a
	}
	return sum / float64(len(v.activity))
}

// Add polarity method to VSIDSHeuristic
func (v *VSIDSHeuristic) GetPreferredPolarity(variable string) bool {
	// Use phase cache if available
	if polarity, exists := v.phaseCache[variable]; exists {
		return polarity
	}

	// Use polarity scores
	if score, exists := v.polarityScores[variable]; exists {
		return score > 0.0
	}

	return true // Default positive
}

func (v *VSIDSHeuristic) rescaleActivities() {
	for variable := range v.activity {
		v.activity[variable] *= 1e-100
	}
	for variable := range v.lrbScores {
		v.lrbScores[variable] *= 1e-100
	}
	v.increment *= 1e-100
}

func (v *VSIDSHeuristic) Reset() {
	v.activity = make(map[string]float64)
	v.lrbScores = make(map[string]float64)
	v.polarityScores = make(map[string]float64)
	v.phaseCache = make(map[string]bool)
	v.participated = make(map[string]int64)
	v.increment = 1.0
	v.decay = 0.95
	v.conflictCount = 0
}

// RandomHeuristic chooses variables randomly (for comparison)
type RandomHeuristic struct{}

func NewRandomHeuristic() *RandomHeuristic {
	return &RandomHeuristic{}
}

func (r *RandomHeuristic) Name() string {
	return "Random"
}

func (r *RandomHeuristic) ChooseVariable(unassigned []string, assignment Assignment) string {
	if len(unassigned) == 0 {
		return ""
	}
	// Simple: return first variable (could use real random)
	return unassigned[0]
}

func (r *RandomHeuristic) Update(conflictClause *Clause) {
	// No update needed for random
}

func (r *RandomHeuristic) Reset() {
	// Nothing to reset
}

// LubyRestartStrategy - now hybrid Luby+Adaptive
type LubyRestartStrategy struct {
	// Core Luby (existing)
	sequence []int
	index    int
	baseUnit int

	// Integrated Glucose-style adaptive restart
	glucoseWindow []int64
	windowSize    int
	windowIndex   int
	fastMA        float64 // Fast moving average
	slowMA        float64 // Slow moving average
	threshold     float64

	// Performance tracking
	restartCount  int64
	lastConflicts int64
}

// NewLubyRestartStrategy - now hybrid with adaptive capabilities
func NewLubyRestartStrategy() *LubyRestartStrategy {
	return &LubyRestartStrategy{
		// Luby sequence (existing)
		sequence: []int{1, 1, 2, 1, 1, 2, 4, 1, 1, 2, 1, 1, 2, 4, 8},
		index:    0,
		baseUnit: 100,

		// Adaptive features enabled by default
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

func (l *LubyRestartStrategy) Name() string {
	return "Luby-Adaptive-Glucose"
}

// Enhanced ShouldRestart with Glucose-style adaptive restart
func (l *LubyRestartStrategy) ShouldRestart(stats SolverStatistics) bool {
	currentConflicts := stats.Conflicts
	recentConflicts := currentConflicts - l.lastConflicts

	// Update moving averages (Glucose-style)
	if recentConflicts > 0 {
		l.glucoseWindow[l.windowIndex] = recentConflicts
		l.windowIndex = (l.windowIndex + 1) % l.windowSize

		alpha := 0.1
		l.fastMA = alpha*float64(recentConflicts) + (1.0-alpha)*l.fastMA
		l.slowMA = 0.01*float64(recentConflicts) + 0.99*l.slowMA
	}

	l.lastConflicts = currentConflicts

	// Glucose-style adaptive restart (primary)
	if l.restartCount > 10 && l.fastMA > l.threshold*l.slowMA {
		return true
	}

	// Luby restart as fallback
	if l.index < len(l.sequence) {
		threshold := int64(l.sequence[l.index] * l.baseUnit)
		return currentConflicts >= threshold
	}

	return false
}

func (l *LubyRestartStrategy) OnRestart() {
	l.restartCount++
	l.index++
	if l.index >= len(l.sequence) {
		l.extendSequence()
	}

	// Adapt threshold based on performance
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
	// Generate more Luby numbers by repeating and extending with a power-of-two term
	current := len(l.sequence)
	for i := 0; i < current; i++ {
		l.sequence = append(l.sequence, l.sequence[i])
	}
	l.sequence = append(l.sequence, int(math.Pow(2, float64(len(l.sequence)))))
}

// ActivityBasedDeletion - now LBD + tier-aware
type ActivityBasedDeletion struct {
	// Core activity (existing)
	activityThreshold float64

	// Integrated LBD awareness
	lbdThreshold  int
	sizeThreshold int

	deletionCount int64
	keepRatio     float64

	// Tier-specific controls
	coreProtection   bool    // never delete core
	midThreshold     float64 // activity threshold for mid-tier
	localThreshold   float64 // activity threshold for local-tier
	recentProtection int64   // recent protection (conflicts), mirrors DB default
}

// NewActivityBasedDeletion - defaults tuned for tiering
func NewActivityBasedDeletion() *ActivityBasedDeletion {
	return &ActivityBasedDeletion{
		activityThreshold: 0.1,
		lbdThreshold:      4,
		sizeThreshold:     30,
		deletionCount:     0,
		keepRatio:         0.5,
		coreProtection:    true,
		midThreshold:      0.15, // slightly higher than base activity
		localThreshold:    0.10, // align with base activity threshold
		recentProtection:  1000, // match DB default
	}
}

func (a *ActivityBasedDeletion) Name() string {
	return "Activity-LBD-Enhanced"
}

// Enhanced ShouldDelete with tier awareness - delegates to ShouldDeleteFromTier
func (a *ActivityBasedDeletion) ShouldDelete(clause *Clause, stats SolverStatistics) bool {
	return a.ShouldDeleteFromTier(clause, clause.Tier, stats)
}

// ShouldDeleteFromTier applies tier-specific strategies
func (a *ActivityBasedDeletion) ShouldDeleteFromTier(clause *Clause, tier int, stats SolverStatistics) bool {
	// Never delete original or unit clauses
	if !clause.Learned || len(clause.Literals) <= 1 {
		return false
	}

	// Core protection or glue/LBD<=2
	if a.coreProtection || clause.Glue || clause.LBD <= 2 || tier == 0 {
		return false
	}

	// Mid-tier (LBD 3-6): careful, activity-based
	if tier == 1 {
		thr := a.midThreshold
		// be a bit less aggressive than plain activity threshold
		if clause.Activity < thr {
			return true
		}
		return false
	}

	// Local tier (LBD > 6): aggressive
	if tier == 2 {
		// Delete if low activity or too large
		if clause.Activity < a.localThreshold || len(clause.Literals) > a.sizeThreshold {
			return true
		}
		// Fallback to generic activity
		return clause.Activity < a.activityThreshold
	}

	// Default fallback
	return clause.Activity < a.activityThreshold
}

// GetDeletionCandidates selects candidates by tiers to reduce DB to its maxSize.
// Priority: Local (aggressive) -> Mid (careful), skip Core and protected Recent.
func (a *ActivityBasedDeletion) GetDeletionCandidates(db *ClauseDatabase, stats SolverStatistics) []*Clause {
	need := db.Size() - db.maxSize
	if need <= 0 {
		return nil
	}

	var out []*Clause

	// Helper to pick from a tier with ShouldDeleteFromTier
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

	// Prefer local, then mid
	_, mid, local, recent := db.GetTierSlices()
	// Skip protected recent by checking bornAt; only promote handles migration out of recent
	_ = recent // kept for completeness; we don't delete from recent directly here

	pick(local, 2)
	if need > 0 {
		pick(mid, 1)
	}

	// Never pick core
	return out
}

// Enhanced Update method with LBD statistics
func (a *ActivityBasedDeletion) Update(clauses []*Clause) {
	if len(clauses) == 0 {
		return
	}

	// Compute LBD and activity statistics
	var lbdSum int
	var clauseCount int
	lbdCounts := make(map[int]int)
	activities := make([]float64, 0, len(clauses))

	for _, clause := range clauses {
		if clause.Learned {
			lbdCounts[clause.LBD]++
			lbdSum += clause.LBD
			clauseCount++
			activities = append(activities, clause.Activity)
		}
	}

	if len(activities) > 0 {
		// Update activity threshold
		sort.Float64s(activities)
		median := activities[len(activities)/2]
		a.activityThreshold = median * 0.3

		// Adapt LBD threshold based on distribution
		avgLBD := float64(lbdSum) / float64(clauseCount)
		if avgLBD < 4.0 {
			// Be more selective when clauses are high quality
			a.lbdThreshold = 3
		} else {
			// Standard threshold
			a.lbdThreshold = 4
		}
	}

	// Adapt parameters based on performance
	a.deletionCount++
	if a.deletionCount%100 == 0 {
		// Adjust keep ratio based on clause quality distribution
		// FIXED: Properly sum lbdCounts[1] and lbdCounts[2] as integers
		glueSum := 0
		if val, ok := lbdCounts[1]; ok {
			glueSum += val
		}
		if val, ok := lbdCounts[2]; ok {
			glueSum += val
		}

		glueRatio := float64(glueSum) / float64(max(1, clauseCount))
		if glueRatio > 0.3 {
			// High quality clauses - be more conservative
			if a.keepRatio < 0.4 {
				a.keepRatio = 0.4
			} else {
				a.keepRatio *= 1.01
			}
		} else {
			// Lower quality clauses - be more aggressive
			if a.keepRatio < 0.3 {
				a.keepRatio = 0.3
			} else {
				a.keepRatio *= 0.99
			}
		}
	}
}

func (a *ActivityBasedDeletion) Reset() {
	a.activityThreshold = 0.1
	a.deletionCount = 0
	a.keepRatio = 0.5
	a.lbdThreshold = 4
}

// Backward compatibility constructors - these now return the enhanced versions

// NewAdvancedVSIDSHeuristic creates enhanced VSIDS (same as NewVSIDSHeuristic now)
func NewAdvancedVSIDSHeuristic() *VSIDSHeuristic {
	return NewVSIDSHeuristic() // All features are now standard
}

// NewAdaptiveRestartStrategy creates enhanced restart strategy (same as NewLubyRestartStrategy now)
func NewAdaptiveRestartStrategy() *LubyRestartStrategy {
	return NewLubyRestartStrategy() // All features are now standard
}

// NewAdvancedClauseDeletion creates enhanced clause deletion (same as NewActivityBasedDeletion now)
func NewAdvancedClauseDeletion() *ActivityBasedDeletion {
	return NewActivityBasedDeletion() // All features are now standard
}

// helper
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
