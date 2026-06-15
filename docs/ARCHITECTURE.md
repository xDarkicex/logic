# SAT Solver Architecture

## Package Overview

The `sat` package implements a state-of-the-art CDCL SAT solver derived from the
Kissat reference architecture. All backing memory is off-heap via
`github.com/xDarkicex/memory` — the Go GC never scans solver data.

```
sat/
├── types.go              Core types: Clause, CNF, Assignment, ClauseDatabase
├── interfaces.go         Solver, Heuristic, RestartStrategy, ConflictAnalyzer
├── cdcl.go               CDCL solver: propagation, decision, backtrack, restart
├── conflict_analysis.go  1st UIP analyzer with LBD computation
├── heuristics.go         VSIDS (LRB, polarity, anti-aging), Luby restarts, deletion
├── var_heap.go           Binary max-heap for O(log n) variable selection
├── trail.go              Arena-backed decision trail with O(1) lookups
├── mode.go               Focused/stable mode switching with reluctant doubling
├── walk.go               WalkSAT pre-solver with phase export
├── inprocessor.go        Vivification, subsumption, BVE, failed literal probing
├── preprocessor.go       Unit propagation, pure literal elimination, subsumption
├── gaussian.go           Gauss-Jordan elimination for XOR constraints
├── cnf_converter.go      Tseitin transformation for all Boolean gates
├── dpll.go               Classic DPLL solver (reference implementation)
├── dpllt.go              DPLL(T) theory solver integration
├── maxsat.go             Weighted MAX-SAT with binary search
├── system.go             SATSystem bridge to logic engine
├── fuzzy_smt.go          Fuzzy SMT: gradient-descent for continuous SAT
└── *_test.go             Unit, integration, and probe tests
```

## Component Diagram

```
                          ┌─────────────────┐
                          │   SATSystemImpl │
                          │   (system.go)   │
                          └────────┬────────┘
                                   │
                    ┌──────────────┼──────────────┐
                    │              │              │
              ┌─────▼─────┐  ┌────▼────┐  ┌──────▼──────┐
              │ CNFConverter│  │  Solver  │  │ MAXSATSolver│
              │ (Tseitin)  │  │(interface)│  │ (maxsat.go) │
              └────────────┘  └────┬─────┘  └─────────────┘
                                   │
                          ┌────────▼────────┐
                          │   CDCLSolver    │
                          │   (cdcl.go)     │
                          └────────┬────────┘
                                   │
          ┌────────────────────────┼────────────────────────┐
          │                        │                        │
    ┌─────▼─────┐          ┌──────▼──────┐          ┌──────▼──────┐
    │ Propagation│          │   Decision   │          │  Backtrack  │
    │ (watched  │          │  (VSIDS heap │          │  (trail.go) │
    │  literals)│          │  + polarity) │          │             │
    └─────┬─────┘          └──────┬──────┘          └─────────────┘
          │                       │
    ┌─────▼─────┐          ┌──────▼──────┐
    │ Conflict  │          │ Mode Switch │
    │ Analysis  │          │ (mode.go)   │
    │ (1st UIP) │          └─────────────┘
    └─────┬─────┘
          │
    ┌─────▼─────┐     ┌──────────┐    ┌───────────┐
    │  Clause   │────▶│ Restart  │───▶│Inprocessing│
    │ Learning  │     │ Strategy │    │(inprocessor│
    └───────────┘     └──────────┘    │    .go)    │
                                      └───────────┘

Pre-solving (optional, runs before CDCL):
    ┌──────────┐     ┌──────────────┐
    │ WalkSAT  │────▶│ Phase Export │──▶ VSIDS polarity cache
    │(walk.go) │     │ to CDCL      │
    └──────────┘     └──────────────┘
```

## Off-Heap Allocation Strategy

The package uses all three allocator types from `github.com/xDarkicex/memory`:

### Pool (`satPool`, `litPool`, `watchPool`)

**Variable-size slices with bulk Reset between solves.**

| Usage | Pool | Typical capacity |
|-------|------|-----------------|
| Clause literal arrays | `litPool` | 3–100 literals |
| General-purpose slices | `satPool` | variable |
| Watch list backing arrays | `watchPool` | per-solver |
| Heap score/pos/stack arrays | `satPool` | N variables |
| WalkSAT counters/unsat/scores | `satPool` | M clauses |
| VSIDS activity arrays | `satPool` | N variables |

Pool is the correct tool for variable-size bulk allocations. All slices allocated
from Pool are zeroed on creation. `Reset()` is never called during a solve —
allocations accumulate and are bulk-released on solver teardown via `Pool.Free()`.

### Arena

**Grow-only data freed on teardown.**

| Usage | Typical capacity |
|-------|-----------------|
| Decision trail entries | 1000+ entries |

The trail uses Arena because it's append-only during search (assignments accumulate)
and freed entirely on solver reset. `ArenaAppend` extends the trail without
switching to the Go heap.

### ShardedFreeList (`clauseAlloc`)

**Fixed-size structs with high allocation churn.**

| Usage | Slot size |
|-------|-----------|
| Clause structs | `unsafe.Sizeof(Clause{})` |

`Clause` structs are fixed-size and allocated/deallocated at high frequency (every
conflict creates learned clauses, GC deletes them). ShardedFreeList is 25.8×
faster than `make()` for 6KB slots and produces zero GC pressure. The slot size
is exactly `unsafe.Sizeof(Clause{})`.

### Why not maps?

`make(map[K]V)` is the sole exception to the off-heap rule — used only for
name→index lookup maps (`varIndex`), small LBD distribution maps (≤ 10 entries),
and the solver's assignment map. All backing slices within maps are Pool-allocated.
There is no off-heap hash table in the allocator.

## Data Flow

### Solve lifecycle

```
Solve(CNF)
  ├── Preprocessor.Preprocess(CNF)          // simplify formula
  ├── WalkSolver.Solve(irredundant)          // local search
  │   └── ExportPhases(VSIDS)                // warm-start if found
  ├── initializeWatchLists()                  // two-watched literals
  ├── initializeHeuristics()                  // VSIDS + heap
  ├── Main CDCL loop:
  │   ├── propagate()                         // watched literals
  │   │   └── propagateXOR()                  // XOR constraints
  │   ├── [if conflict]
  │   │   ├── analyzer.Analyze()              // 1st UIP + LBD
  │   │   ├── learnClause()                   // add to DB
  │   │   ├── backtrack()                     // non-chronological
  │   │   │   └── heuristic.OnBacktrack()     // re-insert vars
  │   │   ├── heuristic.Update()              // bump VSIDS + heap
  │   │   ├── [if should restart]
  │   │   │   ├── restart()
  │   │   │   └── [if should switch mode]
  │   │   │       └── modeSwitcher.Switch()
  │   │   └── [if DB full] deleteClauses()
  │   └── [else]
  │       ├── [if all assigned] return SAT
  │       └── decide()
  │           ├── chooseDecisionVariable()    // heap or reluctant
  │           └── assign()
  └── return result
```

### Variable decision flow

```
chooseDecisionVariable()
  ├── [Stable mode + reluctant trigger] → random pick from unassigned
  └── [Focused mode or non-random stable]
      └── heuristic.ChooseVariable()
          ├── ensureVar() for each unassigned
          ├── heap.Max() to peek
          └── [if assigned] heap.PopMax(), repeat
```

### Conflict analysis flow

```
analyzer.Analyze(conflictClause, trail)
  ├── Initialize learntClause from conflict
  ├── Track current-level variables
  ├── While current-level vars > 1:
  │   ├── findMostRecentVariable()
  │   ├── Get reason clause from trail
  │   ├── resolveWithLBDTracking()
  │   └── countCurrentLevelVars()
  ├── buildLearnedClauseWithLBD()
  └── computeBacktrackLevel()
```

## Key Interfaces

```go
type Solver interface {
    Solve(cnf *CNF) *SolverResult
    SolveWithTimeout(cnf *CNF, timeout time.Duration) *SolverResult
    AddClause(clause *Clause) error
    GetStatistics() SolverStatistics
    Reset()
    Name() string
}

type Heuristic interface {
    ChooseVariable(unassigned []string, assignment Assignment) string
    Update(conflictClause *Clause)
    OnBacktrack(unassigned []string)
    Reset()
    Name() string
}

type RestartStrategy interface {
    ShouldRestart(stats SolverStatistics) bool
    OnRestart()
    Reset()
    Name() string
}

type ConflictAnalyzer interface {
    Analyze(conflictClause *Clause, trail DecisionTrail) (*Clause, int)
    Reset()
    Name() string
}
```

## Testing Strategy

### Unit tests
Each component has dedicated unit tests covering normal operation and edge cases.
~170 tests total, 77% statement coverage, all passing with `-race`.

### Integration tests (`sat_integration_test.go`)
Full CDCL solver runs on satisfiable, unsatisfiable, and corner-case formulas.
Verifies cross-component data flow: preprocessing → WalkSAT → CDCL → result.

### Probe tests (`sat_probe_test.go`)
Directly call unexported functions with crafted internal state to hit code paths
that require thousands of CDCL conflicts to trigger naturally. This is the Go
equivalent of C's probe testing — tests in `package sat` have full access to
unexported symbols. Covers XOR conflict handling, inprocessing execution,
preprocessor internals, DPLL helpers, Gaussian matrix operations, and clause
database management.

## Constants and Tuning

| Parameter | Value | Source |
|-----------|-------|--------|
| VSIDS decay | 0.95 | Chaff/Kissat |
| VSIDS rescale threshold | 1e100 | Kissat |
| VSIDS weight | 0.7 | empirical |
| LRB weight | 0.3 | empirical |
| LRB decay | 0.8 | Glucose |
| Anti-aging threshold | 100 conflicts | empirical |
| LBD glue threshold | ≤ 2 | Glucose |
| Luby base unit | 100 conflicts | MiniSat |
| Glucose adaptive α | 0.1 fast, 0.01 slow | Glucose |
| Mode switch base conflict | 1000 | Kissat |
| Mode switch base tick | 500 | Kissat |
| WalkSAT max flips | 10,000 | empirical |
| WalkSAT score CB | 2.0 | Kissat |
| Max learned clauses | 2000 | empirical |
| Inprocessing gap | 4000 conflicts | Kissat |
| Gaussian frequency | 5000 conflicts | Kissat |
| Clause DB recent protection | 1000 conflicts | empirical |
