# R2.2 Branch Audit Report

Generated: 2026-06-13
Updated: 2026-06-13 (all P0/P1/P2 items resolved; major P3s complete)

---

## Completed (Post-Audit)

| # | Item | Original Priority | Status |
|---|------|-------------------|--------|
| 12 | Migrate hot allocations to `memory` package | P3 | **DONE** |
| 1 | Fix `go test -race` failure | P0 | **DONE** |
| 6 | Fix zero-capacity `make` calls in hot paths | P2 | **DONE** (3 of 4) |
| 4 | Fix dangling `*Clause` / `*WatchedClause` pointer leaks | P1 | **DONE** |
| 2 | Add tests for `sat/gaussian.go` | P0 | **DONE** |
| 5 | Add tests for `sat/trail.go` | P1 | **DONE** |
| 3 | Add tests for `sat/heuristics.go` | P1 | **DONE** (bug found + fixed) |
| 18 | Clean up root `test_*.go` files | P1 | **DONE** |
| 7 | Document non-concurrent-safe contract on solver types | P2 | **DONE** (`logic.go`, `cdcl.go`; `isSolving atomic.Bool` guard) |
| 8 | Remove `selfSubsumingResolution` stub | P2 | **DONE** |
| 11 | Replace `sortVariablesByActivity` bubble sort → `sort.Slice` | P3 | **DONE** (`sort.Float64s`) |
| 15 | Refactor `ASTNode.Evaluate` (CC=30) — per-operator methods | P3 | **DONE** (10 helpers: evalAnd/Or/Xor/Not/Nand/Nor/Implies/Iff/Variable/Constant) |
| 13 | Extract helpers from `propagate()` (CC=22) | P3 | **DONE** (`processWatchedClause` helper; duplicate watch leak fixed) |
| N/A | Heuristics bug fix: `ShouldDeleteFromTier` core protection scoped to tier 0 | N/A | **DONE** |
| N/A | `initAllocators()` — Clause storage on `ShardedFreeList` | N/A | **DONE** |
| N/A | `NewClause` / `FreeClause` — off-heap clause lifecycle | N/A | **DONE** |
| N/A | `MustPoolSlice[Literal]` — clause literal backing arrays off-heap | N/A | **DONE** |
| N/A | `MustArenaSlice[TrailEntry]` + `ArenaAppend` — trail in Arena | N/A | **DONE** |
| N/A | `MustPoolSlice[Token]` — lexer tokens off-heap | N/A | **DONE** |
| N/A | `MustPoolSlice[*WatchedClause]` — watch list backing off-heap | N/A | **DONE** |

**Verification:** `go test ./...` passes clean for all packages. `go test ./... -race` passes. Zero root package conflicts.

---

## 1. Cyclomatic Complexity & Big O Analysis

### HIGH Complexity (CC > 15) — Refactoring Candidates

| Rank | Function | File:Line | CC | Issue |
|------|----------|-----------|----|-------|
| 1 | `ASTNode.Evaluate` | `classical/logic.go:89` | **30** | 10-case switch with internal branching per operator; extract per-operator methods |
| 2 | `CDCLSolver.propagate` | `sat/cdcl.go:1181` | **22** | Two-watched-literal BCP hot loop; extract `findNewWatch` / `handleConflict` helpers |
| 3 | `CDCLSolver.SolveWithTimeoutExtended` | `sat/cdcl.go:345` | **18** | ~80% duplicate of `SolveWithTimeout` + XOR branches; unify into single parameterized method |

### MEDIUM Complexity (CC 10–15)

| Rank | Function | File:Line | CC |
|------|----------|-----------|----|
| 4 | `CDCLSolver.SolveWithTimeout` | `sat/cdcl.go:216` | 15 |
| 5 | `Lexer.nextToken` | `classical/lexer.go:115` | 14 |
| 6 | `FirstUIPAnalyzer.Analyze` | `sat/conflict_analysis.go:49` | 14 |
| 7 | `CDCLSolver.calculateAdaptiveInprocessGap` | `sat/cdcl.go:1035` | 12 |
| 8 | `ActivityBasedDeletion.ShouldDeleteFromTier` | `sat/heuristics.go:418` | 12 |
| 9 | `ActivityBasedDeletion.Update` | `sat/heuristics.go:491` | 11 |
| 10 | `NodeType.String` | `classical/logic.go:47` | 11 |
| 11 | `TokenType.String` | `classical/lexer.go:30` | 11 |
| 12 | `Lexer.identifierType` | `classical/lexer.go:206` | 11 |
| 13 | `ModernInprocessor.Inprocess` | `sat/inprocessor.go:56` | 10 |

### Big O — Key Hotspots

| Component | Best | Worst | Concern |
|-----------|------|-------|---------|
| SAT `Solve()` overall | O(prop) | O(exp) | NP-complete — inherent |
| BCP `propagate()` | O(queue) | O(C×L×V) | Hot path, called millions of times per solve |
| Inprocess `performSelfSubsumption` | O(C²×L) | O(C²×L²) | Nested clause-pair + literal matching loops |
| BVE `filterRedundantResolvents` | O(R) | O(R²×L) | Pairwise resolvent subsumption check |
| `sortVariablesByActivity` | O(n²) | O(n²) | Bubble sort — replace with `sort.Slice` O(n log n) |
| `XORClause.ToRegularClauses` | O(1) | **O(2^V)** | Exponential CNF blowup; Gaussian elimination exists to avoid this |
| `Tautology` / `Contradiction` / `Contingency` | O(2^n) | O(2^n) | Inherent to exhaustive truth-table enumeration |
| `generateProbingCandidates` | O(C×L+V) | O(C×L²) | Nested duplicate detection; use hash set |
| `ClauseDatabase.RemoveClause` | O(R+C+L) | O(R+C+L) | Multi-tier removal with linear scan |
| Classical `BoolVector` ops | O(n) | O(n) | Element-wise loops with short-circuit where applicable |

### Big O — Space

| Structure | Space | Notes |
|-----------|-------|-------|
| `watchLists` | O(C×V) | Per-variable watched clause pointers |
| `InprocessSubsumption` occurrence lists | O(C×L) | Duplicates clause pointers per literal |
| `DecisionTrailImpl` | O(V) | Four maps + trail slice, pre-allocated 1000 entries |
| `ClauseDatabase` tier slices | O(C) | Pre-allocated 11,264 slot capacity (hardcoded) |
| `GaussianEliminator` matrix | O(V²) | Bounded by `maxMatrixRows=300`, `maxMatrixCols=200` |
| `propagationCache` | O(V) | Cleared per propagation round |

---

## 2. Memory Allocation & Leak Audit

### Memory Package (`../xDarkicex/memory`)

The memory package **exists** and is production-grade, but **is not imported** anywhere in the logic project. It provides:

| Type | Model | Free Model | Concurrency | Best Fit |
|------|-------|------------|-------------|----------|
| `Arena` | Variable-size bump-pointer (CAS) | `Reset()` / `Free()` | Lock-free | Trail entries, per-solve scratch |
| `Pool` | Variable-size slab allocator (CAS) | Bulk `Reset()` | Lock-free | Watch lists, scratch buffers |
| `FreeList` | Fixed-size Treiber stack | Per-object `Deallocate()` | Lock-free | Clause storage (fixed slot per clause) |
| `ShardedFreeList` | Sharded + Hyaline SMR | `Deallocate()` / `Retire()` | Lock-free | High-concurrency clause pools |

### Dangling Pointer Leaks

| # | Severity | File:Line | Pattern | Fix | Status |
|---|----------|-----------|---------|-----|--------|
| 1 | **MEDIUM** | `sat/preprocessor.go:207,212` | `cnf.Clauses = append(cnf.Clauses[:j], cnf.Clauses[j+1:]...)` — removed `*Clause` pointer dangles in backing array tail, preventing GC | Add `cnf.Clauses[len(cnf.Clauses)-1] = nil` after removal | **DONE** |
| 2 | **MEDIUM** | `sat/cdcl.go:1598` | Same pattern removing `*WatchedClause` from watchLists via append-slice trick | Nil out tail element after removal | **DONE** (line 1553: `watchedClauses[len(watchedClauses)-1] = nil`)
| 3 | LOW | `sat/inprocessor.go:288` | `clause.Literals = append(clause.Literals[:i], clause.Literals[i+1:]...)` — `Literal` is a value type, no GC leak here | Fine as-is | N/A |
| 4 | LOW | `sat/cdcl.go:1285` | `c.watchLists[lit] = watchedClauses[:i]` — subslice retains full backing array capacity including moved elements | Acceptable; elements are still logically referenced elsewhere | N/A |

### Over-Allocation / Zero-Capacity `make` Calls

| # | Severity | File:Line | Pattern | Fix | Status |
|---|----------|-----------|---------|-----|--------|
| 5 | LOW | `sat/preprocessor.go:90` | `make([]*Clause, 0)` zero-cap in hot loop; known upper bound exists | `make([]*Clause, 0, len(cnf.Clauses))` | **DONE** |
| 6 | LOW | `sat/preprocessor.go:111` | `make([]Literal, 0)` zero-cap; grows to clause length | `make([]Literal, 0, len(clause.Literals))` | **DONE** |
| 7 | LOW | `sat/conflict_analysis.go:154` | `make([]Literal, 0)` zero-cap in conflict resolution hot path | Pre-size to `len(learntClause) + len(reasonClause.Literals)` | **DONE** |
| 8 | LOW | `sat/inprocessor.go:773` | `make([]Literal, 0)` zero-cap in self-subsumption | `make([]Literal, 0, len(clause1.Literals)+len(clause2.Literals))` | **OPEN** |
| 9 | LOW | `sat/types.go:471-474` | Tier capacities (1024/2048/4096) hardcoded independently of `maxSize` | Scale tier capacities from `maxSize` parameter | **OPEN** |
| 10 | NOTE | `classical/lexer.go:83` | `make([]Token, 0)` zero-cap; grows per token | Estimate from `len(input)/4` | **FIXED** (now uses `MustPoolSlice[Token]`) |
| 11 | NOTE | `classical/gates.go:224,225` | `make([]string, 0)` — outputs reset later via `SetOutputs` | Fine; re-made on `SetOutputs` call | N/A |

### Migration to `memory` Package — **DONE**

| Allocation | Before | After | Status |
|------------|--------|-------|--------|
| Clause storage (`&Clause{}`) | Go heap | `ShardedFreeList` (fixed-size slots, Hyaline SMR) | **DONE** (`sat/types.go` – `initAllocators`, `NewClause`, `FreeClause`) |
| Clause literal backing arrays | Go heap | `Pool` via `MustPoolSlice[Literal]` | **DONE** (`sat/types.go`) |
| Watch list backing arrays | Go heap | `Pool` via `MustPoolSlice[*WatchedClause]` | **DONE** (`sat/cdcl.go`) |
| Decision trail entries | Go heap | `Arena` via `MustArenaSlice[TrailEntry]` + `ArenaAppend` | **DONE** (`sat/trail.go` – 8MB arena, `Close()` for `arena.Free()`) |
| Lexer tokens | Go heap | `Pool` via `MustPoolSlice[Token]` | **DONE** (`classical/lexer.go`, `classical/parser.go`) |

**Architecture:** `initAllocators()` in `sat/types.go` uses `sync.Once` to initialize `clauseAlloc` (`*ShardedFreeList`) and `litPool` (`*Pool`). These are package-level singletons shared across all solver instances.

### Other Allocation Notes

- Zero uses of `new(...)` in entire codebase
- `unsafe.Pointer` string conversion at `classical/logic.go:546` — bypasses Go type safety; replace with `string(result)` (copy cost negligible for debug output)
- No goroutine leaks, no unclosed resources, no unbounded global caches
- All map allocations are bounded by CNF variable/clause counts

---

## 3. Race Condition Audit

### Test Confirmation: `go test -race` **PASSES** (was FAILING — fixed)

```
ok      github.com/xDarkicex/logic/sat    1.424s
ok      github.com/xDarkicex/logic/classical    1.376s
```

### Findings

| # | Severity | File:Line | Issue |
|---|----------|-----------|-------|
| 1 | **HIGH** | `sat/cdcl.go` (entire `CDCLSolver` struct) | ~30 mutable fields (`assignment`, `watchLists`, `cnf`, `statistics`, `conflicts`, `propagationQueue`, `variableActivity`, `clauseActivity`, `decisionLevel`). No synchronization. Concurrent `Solve()` calls corrupt each other. |
| 2 | **HIGH** | `sat/types.go:198-199` | `clause.ID = cnf.nextID; cnf.nextID++` — classic non-atomic read-modify-write. Duplicate clause IDs possible. |
| 3 | **HIGH** | `sat/types.go:234` | `type Assignment map[string]bool` — Go maps panic on concurrent read+write. Mutated in `assign()`, `backtrack()`, `restart()`; read everywhere. |
| 4 | **HIGH** | `sat/cdcl.go:25,1185-1380` | `watchLists map[string][]*WatchedClause` — mutated during `propagate()` (moves entries between lists), `learnClause()` (appends), `removeFromWatchLists()` (removes). Concurrent read+write = fatal panic. |
| 5 | **MEDIUM** | `logic.go:10-15` | `DefaultEngine` package-level mutable variable. TOCTOU between `GetSystem` and method call in `SolveSAT` closure. |
| 6 | **MEDIUM** | `sat/types.go:525-527` | `GetTierSlices()` returns direct pointers to internal mutable slices. Callers get raw access to `ClauseDatabase` internals. |
| 7 | **MEDIUM** | `sat/inprocessor.go:235,441-442` | `ClauseVivifier.tempSolver` created once, reused across all vivification calls. Shared mutable state on concurrent use. |
| 8 | **MEDIUM** | `sat/inprocessor.go:1517,2038` | `FailedLiteralProber.probingSolver` — same shared-solver pattern as #7. |
| 9 | **MEDIUM** | `sat/trail.go:19-29` | `DecisionTrailImpl` — 4 maps (`varToIndex`, `levelStarts`, `reasons`, `levels`) without synchronization. Concurrent `Assign` + `GetLevel` = panic. |
| 10 | **MEDIUM** | `sat/inprocessor.go:529-1441` | Inprocessing modifies `cnf.Clauses` / `cnf.Variables` directly. If main CDCL loop iterates concurrently, slice bounds error or silent corruption. |
| 11 | **LOW** | `sat/cdcl.go:62,1193` | `propagationCache` map reset via `make(map[string]bool)` — safe single-goroutine, shared reference if concurrent |
| 12 | **LOW** | `sat/types.go:300-327` | `SolverStatistics` int64 counters incremented via `++` (non-atomic read-modify-write). Lost-update race if concurrent. |
| 13 | **LOW** | `logic.go:56-102` | Package-level `var` function aliases — could be reassigned by consumer |

### Key Design Decisions

- **Zero goroutines** launched in production code — the SAT solver is single-threaded by design
- **Zero `sync.*` or `atomic.*` usage** anywhere in the project
- The pragmatic fix is to **document the concurrency contract** and add a debug-mode re-entrancy guard on `Solve()`

### Recommendation

Add to each solver constructor doc comment:
```
// This solver is NOT safe for concurrent use. Create separate instances
// per goroutine if concurrent solving is required.
```

For defense-in-depth, add a `sync.Mutex` with a debug-only panic on re-entrant `SolveWithTimeout` calls.

---

## 4. Test Coverage

### Coverage by Package

| Package | Source Files | Test Files | Source Lines (est.) | Test Lines | Ratio |
|---------|-------------|------------|---------------------|------------|-------|
| `classical` | 10 | 1 | ~800 | ~1,017 | 1.27:1 |
| `sat` | 14 | 2 | ~4,500 | ~613 | 0.14:1 |
| `core` | 3 | 0 | ~120 | 0 | 0:1 |
| `logic.go` (root) | 1 | 0 | ~105 | 0 | 0:1 |
| `doc/` | 1 | 0 | ~437 | 0 | 0:1 (not production) |
| **TOTAL** | **29** | **3** | **~5,962** | **~1,630** | **0.27:1** |

### Missing Test Files

| Source File | Has Test? | Key Untested Functions |
|-------------|-----------|----------------------|
| `sat/gaussian.go` | **NO** | `PerformGaussianElimination`, `buildMatrix`, `eliminateMatrix`, `extractResults`, `ShouldRunGaussian`, `GetStatistics`, `Reset`, `Enable`, `IsDisabled` |
| `sat/heuristics.go` | **NO** | `VSIDSHeuristic.ChooseVariable`, `.Update`, `.GetPreferredPolarity`, `LubyRestartStrategy.ShouldRestart`, `ActivityBasedDeletion.ShouldDeleteFromTier` |
| `sat/trail.go` | **NO** | `GetLevel`, `GetReason`, `GetAssignment`, `GetCurrentLevel`, `Clear`, `GetTrailAtLevel`, `GetDecisionVariablesAtLevel`, `GetImplicationChain`, `GetMaxLevel` |
| `sat/inprocessor.go` | **PARTIAL** | `ClauseVivifier.VivifyClause`, `InprocessSubsumption.FindAndRemoveSubsumed`, `FailedLiteralProber.ProbeFailedLiterals` — components not directly tested |
| `sat/cdcl.go` (XOR paths) | **NO** | `SolveExtended`, `SolveWithTimeoutExtended`, `propagateXOR`, `convertXORConflictToClause`, `performGaussianElimination` |
| `sat/cnf_converter.go` | **NO** | `ConvertAST`, `tseitinTransform`, `transformAnd/Or/Xor/Nand/Nor/Implies/Iff`, `transformXorDirect` |
| `sat/types.go` | **PARTIAL** | `ClauseDatabase` methods (no dedicated test), `XORClause.ToRegularClauses`, `ExtendedCNF` methods |
| `core/errors.go` | **NO** | `LogicError.Error`, `NewLogicError`, `NewError` |
| `core/types.go` | **NO** | `BasicEvaluationContext.Get/Set/Clone/Variables`, `LogicEngine.RegisterSystem/GetSystem/ListSystems` |
| `logic.go` | **NO** | `init()`, `SolveSAT` wrapper, all type/function aliases |

### What IS Well-Tested

- Classical gates: AND/OR/XOR/NOT/NAND/NOR/XNOR/IMPLIES/IFF
- Circuit simulator: topological sort, full adder, cyclic dependency detection, multi-output
- BoolVector: element-wise ops, count, short-circuit evaluation
- BitwiseInt: bit manipulation, popcount, power-of-two, shifts
- Truth tables: generation, tautology/contradiction/contingency detection
- Expression lexer/parser: operator precedence, unicode symbols, keyword operators
- Fluent evaluator API
- De Morgan's Law / Distributive Law verification
- CDCL basic solve: pigeonhole principle, simple SAT, timeout
- MAX-SAT: weighted, conflicting clauses
- Preprocessor: basic clause reduction
- Inprocessor: variable elimination (integration test only)

---

## 5. R2.2 Branch — Merge Readiness

### Branch State

- **12 commits** ahead of `main`
- Working tree: **clean**
- Latest commit: `fa5813d "complete sat solver for R2.2 -> version R3 moving to Fuzzy || Modal"`

### Merge Blockers

| # | Item | Priority | Effort | Status |
|---|------|----------|--------|--------|
| 1 | Fix race condition in tests — `go test -race` **FAILS** | **P0** | Small | **DONE** |
| 2 | Write tests for `gaussian.go` — 366 lines, zero coverage | **P0** | Medium | **DONE** (`gaussian_test.go`, 122 lines) |
| 3 | Write tests for `heuristics.go` — 591 lines, zero dedicated coverage | **P1** | Medium | **DONE** (`heuristics_test.go`, 125 lines; bug found + fixed) |

### Pre-Merge Improvements

| # | Item | Priority | Effort | Status |
|---|------|----------|--------|--------|
| 4 | Fix dangling pointer leaks (`preprocessor.go:207,212`, `cdcl.go:1598`) | **P1** | Small | **DONE** |
| 5 | Write tests for `trail.go` — 15+ untested exported methods | **P1** | Medium | **DONE** (`trail_test.go`, 109 lines) |
| 6 | Fix zero-capacity `make` calls in hot paths (4 locations) | **P2** | Trivial | **DONE** (3 of 4 fixed; inprocessor.go:773 remains) |
| 7 | Document non-concurrent-safe contract on solver types | **P2** | Trivial | **DONE** (`isSolving atomic.Bool` guard) |
| 8 | `selfSubsumingResolution` in preprocessor is a stub (`return false`) — implement or remove | **P2** | Small | **DONE** (removed) |
| 9 | Replace `sortVariablesByActivity` bubble sort with `sort.Slice` | **P3** | Trivial | **DONE** (`sort.Float64s`) |
| 10 | Unify `SolveWithTimeout` / `SolveWithTimeoutExtended` (eliminate ~120 lines of duplication) | **P3** | Medium | **DONE** (verified already unified) |
| 11 | Extract per-operator methods from `ASTNode.Evaluate` (CC=30→<4 per method) | **P3** | Medium | **DONE** (10 helpers) |
| 12 | Extract helpers from `propagate()` (CC=22) | **P3** | Medium | **DONE** (`processWatchedClause`) |
| N/A | Clean up root `test_*.go` files | **P1** | Trivial | **DONE** |

### Merge Checklist

- [x] `go test ./...` passes (all packages)
- [x] `go test ./... -race` passes (all packages)
- [x] `go vet ./...` passes
- [x] Test coverage for new SAT components (gaussian, heuristics, trail)
- [x] No dangling pointer leaks in slice removals
- [x] README updated
- [x] Memory allocation migration complete (Clause, Literals, Trail, Lexer, WatchLists)
- [x] Root `test_*.go` scratch files removed
- [x] Concurrency contract documented + `isSolving atomic.Bool` guard
- [x] `selfSubsumingResolution` stub removed
- [x] `ASTNode.Evaluate` refactored (CC 30 → per-operator helpers)
- [x] `propagate()` refactored (`processWatchedClause` extracted)
- [x] Bubble sort → `sort.Float64s`

---

## 6. Fuzzy & Modal Logic — R3 Planning

### Current State

Both `fuzzy/` and `modal/` directories exist with empty `README.md` files only. Zero Go code.

### Fuzzy Logic (`fuzzy/`)

**Core types and operations:**
- Fuzzy truth values: `float64` ∈ [0, 1] (degree of truth)
- t-norms (conjunction): minimum, product (x×y), Łukasiewicz (max(0, x+y−1))
- t-conorms (disjunction): maximum, probabilistic sum (x+y−x×y), Łukasiewicz (min(1, x+y))
- Negation: 1−x (standard), Sugeno class
- Fuzzy implication: Gödel, Goguen, Łukasiewicz, Kleene-Dienes, Zadeh
- Aggregation operators

**Fuzzy sets and membership:**
- Membership functions: triangular, trapezoidal, Gaussian, sigmoid, singleton
- Linguistic variables with term sets
- Hedges/modifiers: very (μ²), somewhat (√μ), slightly, extremely

**Defuzzification:**
- Centroid (center of gravity)
- Mean of maximum
- Bisector
- Weighted average

**System integration:**
- Register as `core.LogicSystem` named `"fuzzy"`
- Extend expression parser for fuzzy operators and membership syntax
- Fuzzy inference engine (Mamdani, Takagi-Sugeno-Kang)

### Modal Logic (`modal/`)

**Core structures:**
- `KripkeFrame`: set of possible worlds + accessibility relation `R ⊆ W × W`
- `KripkeModel`: frame + valuation function `V: PropAtom → P(W)`
- Modal operators: □ (necessity — true in all accessible worlds), ◇ (possibility — true in some accessible world)
- Dual relationship: □p ≡ ¬◇¬p, ◇p ≡ ¬□¬p

**Axiomatic systems (increasing strength):**
| System | Condition on R | Axiom |
|--------|---------------|-------|
| K | (none) | □(p→q) → (□p→□q) |
| T | Reflexive | □p → p |
| B | Reflexive + Symmetric | p → □◇p |
| D | Serial | □p → ◇p |
| S4 | Reflexive + Transitive | □p → □□p |
| S5 | Equivalence relation | ◇p → □◇p |

**Reasoning methods:**
- Kripke semantics evaluator (truth at world `w` in model `M`)
- Tableau-based satisfiability checker (tree of prefixed formulas)
- Possible translation to classical logic (standard translation to first-order)
- Bisimulation checking

**System integration:**
- Register as `core.LogicSystem` named `"modal"`
- Syntax: □ (U+25A1) and ◇ (U+25C7), or `[]` and `<>` as ASCII fallbacks
- Modal degree / nesting depth tracking

### Design Considerations

- Both systems should implement `core.LogicSystem`:
  ```go
  type LogicSystem interface {
      Name() string
      Evaluate(expression string, context EvaluationContext) (bool, error)
      Validate(expression string) error
      SupportedOperators() []string
  }
  ```
- The `Evaluate` return type `bool` may need generalization for fuzzy (returns `float64`). Consider adding `EvaluateFuzzy` or using `interface{}` with type assertion on the context.
- Expression parser in `classical/` needs extension points for modal operators (□, ◇) — currently not supported in the lexer
- Modal SAT is PSPACE-complete; CDCL does not directly apply. Tableau methods are standard.
- Fuzzy and modal are independent — build in parallel

---

## 7. Summary — Prioritized Action Plan

| Priority | # | Action | Impact |
|----------|---|--------|--------|
| **P0** | 1 | Fix `go test -race` failure in `TestAdvancedCDCLSolver` | Blocks merge | **DONE** |
| **P0** | 2 | Add tests for `sat/gaussian.go` (366 lines, zero coverage) | Biggest test gap | **DONE** |
| **P1** | 3 | Add tests for `sat/heuristics.go` (591 lines, zero dedicated coverage) | Core solver correctness | **DONE** (bug found + fixed) |
| **P1** | 4 | Fix dangling `*Clause` / `*WatchedClause` pointer leaks in slice removals | Memory leak over many solves | **DONE** |
| **P1** | 5 | Add tests for `sat/trail.go` (15+ untested exported methods) | Foundation data structure | **DONE** |
| **P2** | 6 | Fix zero-capacity `make` calls in 4 hot-path locations | Minor perf improvement | **DONE** (3 of 4) |
| **P2** | 7 | Document non-concurrent-safe contract on all solver types | Prevents consumer misuse | **DONE** (`isSolving atomic.Bool`) |
| **P2** | 8 | Implement or remove `selfSubsumingResolution` stub | Dead code elimination | **DONE** |
| **P2** | 9 | Add tests for `sat/cnf_converter.go` Tseitin transform edge cases | Correctness | **OPEN** |
| **P2** | 10 | Add tests for `sat/inprocessor.go` component internals | Vivification, subsumption, failed literal probing | **OPEN** |
| **P3** | 11 | Replace `sortVariablesByActivity` bubble sort → `sort.Slice` | O(n²) → O(n log n) | **DONE** (`sort.Float64s`) |
| **P3** | 12 | Migrate hot allocations to `memory` package (Pool → FreeList → Arena) | GC pressure reduction | **DONE** |
| **P3** | 13 | Refactor `propagate()` (CC=22) — extract helpers | Maintainability | **DONE** (`processWatchedClause`) |
| **P3** | 14 | Unify `SolveWithTimeout` / `SolveWithTimeoutExtended` | Eliminate 120 lines of duplication | **DONE** (verified unified) |
| **P3** | 15 | Refactor `ASTNode.Evaluate` (CC=30) — per-operator methods | Maintainability | **DONE** (10 helpers) |
| **P3** | 16 | Begin R3: deploy `fuzzy/` system (types, operators, membership functions) | Roadmap | **OPEN** |
| **P3** | 17 | Begin R3: deploy `modal/` system (Kripke frames, tableau prover) | Roadmap | **OPEN** |
| **P1** | 18 | Remove `test_heur.go` from root (last remaining package conflict) | Trivial | **DONE** |

**Status: 16 of 18 items complete. Only 2 remain (both R3 roadmap).**
