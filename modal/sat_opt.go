package modal

import "github.com/xDarkicex/memory"

// ── B.2 Acceptance-Driven Clause Learning ───────────────────────────

// DeadSubspaceTracker marks decision-level assignment subspaces as "dead"
// after they are fully explored and proven unsatisfiable. This prevents
// redundant exploration across restarts in a CDCL SAT solver.
//
// A subspace is identified by the set of variables assigned at a given
// decision level. Once marked dead, future restarts skip that subspace.
type DeadSubspaceTracker struct {
	dead   []bool // dead[v] = true if variable v's assignment subspace is exhausted
	pool   *memory.Pool
}

// NewDeadSubspaceTracker creates a tracker for n variables.
func NewDeadSubspaceTracker(n int, pool *memory.Pool) *DeadSubspaceTracker {
	dead := memory.MustPoolSlice[bool](pool, n)
	dead = dead[:n]
	return &DeadSubspaceTracker{dead: dead, pool: pool}
}

// MarkDead marks the given variable subset as exhausted.
func (dt *DeadSubspaceTracker) MarkDead(vars []int32) {
	for _, v := range vars {
		if int(v) < len(dt.dead) {
			dt.dead[v] = true
		}
	}
}

// IsDead returns true if any variable in the set is marked dead.
func (dt *DeadSubspaceTracker) IsDead(vars []int32) bool {
	for _, v := range vars {
		if int(v) < len(dt.dead) && dt.dead[v] {
			return true
		}
	}
	return false
}

// Reset clears all dead markings for a fresh search.
func (dt *DeadSubspaceTracker) Reset() {
	for i := range dt.dead {
		dt.dead[i] = false
	}
}

// DeadCount returns the number of dead-marked variables.
func (dt *DeadSubspaceTracker) DeadCount() int {
	n := 0
	for _, d := range dt.dead {
		if d {
			n++
		}
	}
	return n
}

// ── B.3 Formula Structure-Guided Variable Ordering ──────────────────

// VariableWeighter assigns activity scores to variables based on modal
// formula structure. Variables from eventual subformulas (◇, U) get
// higher weight — they are more likely to be decision-critical.
// Variables from universal subformulas (□) get moderate weight.
// Boolean-only variables get baseline weight.
type VariableWeighter struct {
	weights []float64 // per-variable activity weight
	pool    *memory.Pool
}

// NewVariableWeighter creates a weighter for n variables.
func NewVariableWeighter(n int, pool *memory.Pool) *VariableWeighter {
	w := memory.MustPoolSlice[float64](pool, n)
	w = w[:n]
	for i := range w {
		w[i] = 1.0 // baseline
	}
	return &VariableWeighter{weights: w, pool: pool}
}

// WeightByClass adjusts variable weights based on formula classification.
// eventual vars get factor 4.0, universal vars get factor 2.0, others stay 1.0.
// CC=4.
func (vw *VariableWeighter) WeightByClass(vars []int32, class FormulaClass) {
	factor := 1.0
	switch class {
	case ClassSuspendable:
		factor = 4.0 // □◇p type: both eventual and universal → highest priority
	case ClassObligation:
		factor = 1.0 // safety: baseline
	default:
		factor = 2.0 // general (includes U, ◇): moderate
	}
	for _, v := range vars {
		if int(v) < len(vw.weights) {
			vw.weights[v] *= factor
		}
	}
}

// WeightByProps adjusts weights using the LTL property flags computed
// by the classifier. CC=4.
func (vw *VariableWeighter) WeightByProps(vars []int32, obl, ev, un bool) {
	factor := 1.0
	if ev && un {
		factor = 4.0 // suspendable: □◇p
	} else if ev {
		factor = 3.0 // eventual: ◇p, pUq
	} else if un {
		factor = 2.0 // universal: □p
	}
	for _, v := range vars {
		if int(v) < len(vw.weights) {
			vw.weights[v] *= factor
		}
	}
}

// Weight returns the current weight for a variable.
func (vw *VariableWeighter) Weight(v int32) float64 {
	if int(v) < len(vw.weights) {
		return vw.weights[v]
	}
	return 1.0
}

// TopN returns the indices of the n highest-weighted variables. CC=6.
func (vw *VariableWeighter) TopN(n int) []int32 {
	if n > len(vw.weights) {
		n = len(vw.weights)
	}
	result := memory.MustPoolSlice[int32](vw.pool, n)
	result = result[:n]
	// Simple selection: build index array and sort by weight.
	indices := memory.MustPoolSlice[int32](vw.pool, len(vw.weights))
	indices = indices[:len(vw.weights)]
	for i := range indices {
		indices[i] = int32(i)
	}
	// Partial selection sort for top n.
	for i := 0; i < n; i++ {
		best := i
		for j := i + 1; j < len(indices); j++ {
			if vw.weights[indices[j]] > vw.weights[indices[best]] {
				best = j
			}
		}
		indices[i], indices[best] = indices[best], indices[i]
		result[i] = indices[i]
	}
	return result
}

// ── B.4 Independent Component Decomposition ─────────────────────────

// ComponentDecomposer splits a set of clauses (represented as variable
// index lists) into independent connected components. Clauses sharing
// variables are in the same component. Each component can be solved
// independently — if any is UNSAT, the whole formula is UNSAT.
type ComponentDecomposer struct {
	pool *memory.Pool
}

// NewComponentDecomposer creates a decomposer.
func NewComponentDecomposer(pool *memory.Pool) *ComponentDecomposer {
	return &ComponentDecomposer{pool: pool}
}

// Decompose splits clauses into independent variable components.
// Each clause is a list of variable indices (positive = literal, negative ignored for connectivity).
// Returns one component per connected group. CC=7.
func (cd *ComponentDecomposer) Decompose(clauses [][]int32) [][][]int32 {
	if len(clauses) <= 1 {
		return [][][]int32{clauses}
	}

	// Find max variable to size the graph.
	maxVar := int32(-1)
	for _, cl := range clauses {
		for _, v := range cl {
			av := v
			if av < 0 {
				av = -av
			}
			if av > maxVar {
				maxVar = av
			}
		}
	}
	if maxVar < 0 {
		return nil
	}

	// Build adjacency: clause connects all its variables.
	n := int(maxVar) + 1
	adj := memory.MustPoolSlice[[]int32](cd.pool, n)
	adj = adj[:n]
	for i := range adj {
		adj[i] = memory.MustPoolSlice[int32](cd.pool, 0)
	}
	for _, cl := range clauses {
		for i := 0; i < len(cl); i++ {
			for j := i + 1; j < len(cl); j++ {
				vi := cl[i]
				vj := cl[j]
				if vi < 0 {
					vi = -vi
				}
				if vj < 0 {
					vj = -vj
				}
				if vi == vj {
					continue
				}
				adj[vi] = append(adj[vi], vj)
				adj[vj] = append(adj[vj], vi)
			}
		}
	}

	// Find connected components among variables that appear in any clause.
	present := memory.MustPoolSlice[bool](cd.pool, n)
	present = present[:n]
	for _, cl := range clauses {
		for _, v := range cl {
			av := v
			if av < 0 {
				av = -av
			}
			present[av] = true
		}
	}

	visited := memory.MustPoolSlice[bool](cd.pool, n)
	visited = visited[:n]
	var comps [][]int32
	for v := int32(0); v < int32(n); v++ {
		if !present[v] || visited[v] {
			continue
		}
		comp := cd.dfsComponent(v, adj, visited)
		comps = append(comps, comp)
	}

	if len(comps) <= 1 {
		return [][][]int32{clauses}
	}

	// Assign each clause to its component.
	result := memory.MustPoolSlice[[][]int32](cd.pool, len(comps))
	result = result[:len(comps)]
	for i := range result {
		result[i] = memory.MustPoolSlice[[]int32](cd.pool, 0)
	}
	// Map variable → component index.
	varComp := memory.MustPoolSlice[int32](cd.pool, n)
	varComp = varComp[:n]
	for i := range varComp {
		varComp[i] = -1
	}
	for ci, comp := range comps {
		for _, v := range comp {
			varComp[v] = int32(ci)
		}
	}
	for _, cl := range clauses {
		// Find component for the first variable in the clause.
		ci := int32(-1)
		for _, v := range cl {
			av := v
			if av < 0 {
				av = -av
			}
			if varComp[av] >= 0 {
				ci = varComp[av]
				break
			}
		}
		if ci >= 0 {
			result[ci] = append(result[ci], cl)
		}
	}
	return result
}

// dfsComponent collects a connected component via DFS. CC=3.
func (cd *ComponentDecomposer) dfsComponent(start int32, adj [][]int32, visited []bool) []int32 {
	result := memory.MustPoolSlice[int32](cd.pool, len(adj))
	result = result[:0]
	stack := memory.MustPoolSlice[int32](cd.pool, len(adj))
	stack = stack[:0]
	stack = append(stack, start)
	visited[start] = true

	for len(stack) > 0 {
		v := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		result = append(result, v)
		for _, u := range adj[v] {
			if !visited[u] {
				visited[u] = true
				stack = append(stack, u)
			}
		}
	}
	return result
}

// flattenClauses collects all clause variable lists into one. CC=2.
func (cd *ComponentDecomposer) flattenClauses(clauses [][]int32) []int32 {
	n := 0
	for _, cl := range clauses {
		n += len(cl)
	}
	result := memory.MustPoolSlice[int32](cd.pool, n)
	result = result[:0]
	for _, cl := range clauses {
		result = append(result, cl...)
	}
	return result
}
