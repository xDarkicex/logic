package sat

import (
	"fmt"

	"github.com/xDarkicex/memory"
)

// TheoryPlugin checks modal theory consistency over Boolean assignments.
// When the SAT solver finds a candidate model, each registered plugin
// validates it. If inconsistent, the plugin returns a theory lemma
// (a clause over the Boolean abstraction) that rules out the bad assignment.
//
// This implements the DPLL(T) architecture: SAT provides Boolean skeletons,
// theory plugins provide domain-specific consistency checks.
//
// Literal encoding: var*2 = positive, var*2+1 = negative.
// To decode: variable = lit/2; negated = lit%2 == 1.
type TheoryPlugin interface {
	// Check validates the current Boolean assignment against the theory.
	// Returns (consistent, conflictClause).
	// conflictClause uses the var*2 / var*2+1 encoding.
	Check(assign []int8) (consistent bool, conflictClause []int32)

	// Name returns the plugin identifier for debugging.
	Name() string
}

// TheorySolver wraps a CDCL SAT solver with integer-indexed variables
// and theory plugin support. All internal arrays use Pool backing.
// Variable indices are dense [0..numVars), matching BDD variable indices
// from the BDDCtx bridge (§B.1).
type TheorySolver struct {
	cdcl    *CDCLSolver
	plugins []TheoryPlugin

	varNames []string // varNames[i] = variable name string
	numVars  int32
	cnf      *CNF
	pool     *memory.Pool
}

// NewTheorySolver creates a DPLL(T) solver with n integer-indexed variables.
func NewTheorySolver(n int, pool *memory.Pool) *TheorySolver {
	names := memory.MustPoolSlice[string](pool, n)
	names = names[:n]
	for i := range names {
		names[i] = fmt.Sprintf("v%d", i)
	}
	return &TheorySolver{
		cdcl:     NewCDCLSolver(),
		varNames: names,
		numVars:  int32(n),
		cnf:      NewCNF(),
		pool:     pool,
	}
}

// AddClause adds a clause from integer literals (positive=var, negative=¬var).
func (ts *TheorySolver) AddClause(lits []int32) {
	ts.cnf.AddClause(ts.litsToClause(lits))
}

// RegisterPlugin adds a theory plugin. Plugins are checked in order after
// each Boolean model is found.
func (ts *TheorySolver) RegisterPlugin(p TheoryPlugin) {
	ts.plugins = append(ts.plugins, p)
}

// NumVars returns the number of Boolean variables.
func (ts *TheorySolver) NumVars() int32 { return ts.numVars }

// Solve attempts to find a satisfying assignment.
// Returns the satisfying assignment (nil if UNSAT) and status. CC=7.
// Each theory lemma causes a fresh SAT call since the CDCL solver
// reinitializes on each Solve; the accumulated CNF captures all lemmas.
func (ts *TheorySolver) Solve() ([]int8, bool) {
	for {
		result := ts.cdcl.Solve(ts.cnf)
		if !result.Satisfiable {
			return nil, false
		}
		assign := ts.assignmentToInts(result.Assignment)
		if ts.checkPlugins(assign) {
			return assign, true
		}
	}
}

// checkPlugins runs all theory plugins against the assignment.
// Returns true if all plugins are satisfied. CC=4.
func (ts *TheorySolver) checkPlugins(assign []int8) bool {
	for _, p := range ts.plugins {
		ok, lemma := p.Check(assign)
		if !ok {
			cl := ts.litsToClause(lemma)
			ts.cnf.AddClause(cl)
			return false
		}
	}
	return true
}

// litsToClause converts integer literals to a Clause via ShardedFreeList.
// Encoding: var*2 = positive, var*2+1 = negative. CC=3.
func (ts *TheorySolver) litsToClause(lits []int32) *Clause {
	literals := memory.MustPoolSlice[Literal](ts.pool, len(lits))
	literals = literals[:len(lits)]
	for i, enc := range lits {
		v := enc / 2
		neg := enc%2 == 1
		literals[i] = Literal{Variable: ts.varNames[v], Negated: neg}
	}
	return NewClause(literals...)
}

// assignmentToInts converts the string-based assignment to []int8. CC=4.
func (ts *TheorySolver) assignmentToInts(assign Assignment) []int8 {
	result := memory.MustPoolSlice[int8](ts.pool, int(ts.numVars))
	result = result[:ts.numVars]
	// Default unassigned to 0 (false) — SAT solver may leave don't-cares unset.
	for i := range result {
		result[i] = 0
	}
	for v, val := range assign {
		idx := ts.varIndex(v)
		if idx >= 0 && int(idx) < len(result) {
			if val {
				result[idx] = 1
			} else {
				result[idx] = 0
			}
		}
	}
	return result
}

// varIndex returns the integer index for a variable name, or -1. CC=2.
func (ts *TheorySolver) varIndex(name string) int32 {
	if len(name) < 2 || name[0] != 'v' {
		return -1
	}
	var idx int32
	for i := 1; i < len(name); i++ {
		idx = idx*10 + int32(name[i]-'0')
	}
	if idx < ts.numVars {
		return idx
	}
	return -1
}

// Reset clears the solver for reuse. CC=2.
func (ts *TheorySolver) Reset() {
	ts.cdcl.Reset()
	ts.cnf = NewCNF()
	ts.plugins = ts.plugins[:0]
}
