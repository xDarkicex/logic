package sat

import (
	"math"
	"sort"
)

// VSIDSHeuristic implements Variable State Independent Decaying Sum
type VSIDSHeuristic struct {
	activity  map[string]float64
	increment float64
	decay     float64
}

// NewVSIDSHeuristic creates a new VSIDS heuristic
func NewVSIDSHeuristic() *VSIDSHeuristic {
	return &VSIDSHeuristic{
		activity:  make(map[string]float64),
		increment: 1.0,
		decay:     0.95,
	}
}

func (v *VSIDSHeuristic) Name() string {
	return "VSIDS"
}

// ChooseVariable selects variable with highest activity
func (v *VSIDSHeuristic) ChooseVariable(unassigned []string, assignment Assignment) string {
	if len(unassigned) == 0 {
		return ""
	}

	// Initialize activities for new variables
	for _, variable := range unassigned {
		if _, exists := v.activity[variable]; !exists {
			v.activity[variable] = 0.0
		}
	}

	// Find variable with maximum activity
	maxActivity := -1.0
	var chosen string

	for _, variable := range unassigned {
		activity := v.activity[variable]
		if activity > maxActivity {
			maxActivity = activity
			chosen = variable
		}
	}

	return chosen
}

// Update bumps activity of variables in conflict clause
func (v *VSIDSHeuristic) Update(conflictClause *Clause) {
	for _, lit := range conflictClause.Literals {
		v.activity[lit.Variable] += v.increment
	}

	// Decay activities
	v.increment /= v.decay

	// Rescale if activities get too large
	if v.increment > 1e100 {
		v.rescaleActivities()
	}
}

func (v *VSIDSHeuristic) rescaleActivities() {
	for variable := range v.activity {
		v.activity[variable] *= 1e-100
	}
	v.increment *= 1e-100
}

func (v *VSIDSHeuristic) Reset() {
	v.activity = make(map[string]float64)
	v.increment = 1.0
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

// LubyRestartStrategy implements Luby sequence for restarts
type LubyRestartStrategy struct {
	sequence []int
	index    int
	baseUnit int
}

func NewLubyRestartStrategy() *LubyRestartStrategy {
	return &LubyRestartStrategy{
		sequence: []int{1, 1, 2, 1, 1, 2, 4, 1, 1, 2, 1, 1, 2, 4, 8},
		index:    0,
		baseUnit: 100,
	}
}

func (l *LubyRestartStrategy) Name() string {
	return "Luby"
}

func (l *LubyRestartStrategy) ShouldRestart(stats SolverStatistics) bool {
	if l.index >= len(l.sequence) {
		// Extend sequence if needed
		l.extendSequence()
	}

	threshold := int64(l.sequence[l.index] * l.baseUnit)
	return stats.Conflicts >= threshold
}

func (l *LubyRestartStrategy) OnRestart() {
	l.index++
	if l.index >= len(l.sequence) {
		l.extendSequence()
	}
}

func (l *LubyRestartStrategy) Reset() {
	l.index = 0
}

func (l *LubyRestartStrategy) extendSequence() {
	// Generate more Luby numbers
	current := len(l.sequence)
	for i := 0; i < current; i++ {
		l.sequence = append(l.sequence, l.sequence[i])
	}
	l.sequence = append(l.sequence, int(math.Pow(2, float64(len(l.sequence)))))
}

// ActivityBasedDeletion deletes clauses with low activity
type ActivityBasedDeletion struct {
	activityThreshold float64
}

func NewActivityBasedDeletion() *ActivityBasedDeletion {
	return &ActivityBasedDeletion{
		activityThreshold: 0.1,
	}
}

func (a *ActivityBasedDeletion) Name() string {
	return "ActivityBased"
}

func (a *ActivityBasedDeletion) ShouldDelete(clause *Clause, stats SolverStatistics) bool {
	return clause.Learned && clause.Activity < a.activityThreshold
}

func (a *ActivityBasedDeletion) Update(clauses []*Clause) {
	// Update activity threshold based on median activity
	if len(clauses) > 0 {
		activities := make([]float64, len(clauses))
		for i, clause := range clauses {
			activities[i] = clause.Activity
		}
		sort.Float64s(activities)

		median := activities[len(activities)/2]
		a.activityThreshold = median * 0.5
	}
}

func (a *ActivityBasedDeletion) Reset() {
	a.activityThreshold = 0.1
}
