# logic

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/xDarkicex/logic)](https://goreportcard.com/report/github.com/xDarkicex/logic)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/xDarkicex/logic.svg)](https://pkg.go.dev/github.com/xDarkicex/logic)
[![Tests](https://img.shields.io/badge/tests-670+-blue.svg)]()
[![Coverage](https://img.shields.io/badge/coverage-77--100%25-brightgreen.svg)]()
[![Race](https://img.shields.io/badge/race-clean-brightgreen.svg)]()

**A complete reasoning engine in pure Go.** Classical propositional logic, a Kissat-derived
CDCL SAT solver, modal logic with Kripke semantics and LTL/epistemic reasoning, and
full fuzzy inference — all off-heap, all zero `make([]T)`, all race-clean.

670+ tests. 77–100% coverage across 8 packages. The Go GC never scans reasoning
data. Built for formal verification, agentic reasoning, and embedded inference
at predictable latency.

---

## Contents

- [Why logic](#why-logic)
- [What we solve](#what-we-solve)
- [Quick start](#quick-start)
- [Packages](#packages)
- [SAT solver](#sat-solver)
- [Modal logic](#modal-logic)
- [Fuzzy logic](#fuzzy-logic)
- [Why xDarkicex/memory](#why-xdarkicexmemory)
- [Why xDarkicex/gobdd](#why-xdarkicexgobdd)
- [Architecture](#architecture)
- [Research backing](#research-backing)
- [When to use / not use](#when-to-use)
- [Links](#links)

---

## Why logic

Most logic libraries pick one paradigm and stop. Real systems — formal verification, agentic AI, embedded controllers, regulatory compliance — need to reason across **classical**, **SAT**, **modal**, and **fuzzy** domains simultaneously. You should not need four libraries, three memory models, and GC pressure to make that happen.

This package gives you all four reasoning paradigms behind one interface, backed by one off-heap memory model, at predictable latency.

| Domain | What you can prove |
|--------|-------------------|
| **Classical** | Truth tables, tautology/contradiction detection, all Boolean gates, DeMorgan/Distributive laws |
| **SAT** | Satisfiability of arbitrary CNF formulas, XOR (parity) constraints, MAX-SAT optimization |
| **Modal** | Validity in K/D/T/B/S4/S5, temporal inevitability (LTL), multi-agent knowledge (epistemic), bisimulation equivalence |
| **Fuzzy** | Continuous-valued inference (Mamdani + TSK), clustering (FCM), neuro-adaptive training (ANFIS), Type-2 uncertainty |

---

## What we solve

**Classical.** Prove `(A → B) ∧ (B → C) → (A → C)` is a tautology. Truth tables,
fluent evaluator, bitwise operations, DeMorgan and Distributive law verification.
Expression parser with all 7 gates plus implication and biconditional.

**SAT.** Find an assignment satisfying 10,000 Boolean constraints with XOR parity
requirements — or prove none exists. 1st UIP clause learning with LBD quality scoring.
Binary max-heap VSIDS for O(log n) variable selection. Gaussian elimination over GF(2)
for native XOR. WalkSAT pre-solving with phase warm-start. Inprocessing (BVE,
subsumption, vivification) between restarts.

**Modal.** Kripke semantics with accessibility relations. Couvreur on-the-fly SCC
emptiness for temporal logic. Multi-agent epistemic reasoning with K/D/T/B/S4/S5
axiom enforcement. Bisimulation contraction with BDD-refined partition signatures.
State-space reduction: stutter invariance, Hopcroft DFA minimization, world-set
encoding (O(1) bit-vector membership). Zielonka parity games for reactive synthesis.

**Temporal.** LTL with □ (always), ◇ (eventually), U (until), ○ (next). Tableau
prover with level-based degeneralization. Now/Next/Promise decomposition. LTL
splitting into obligation/suspendable/rest components. Cut-point relabeling via
articulation points.

**Fuzzy.** Mamdani and TSK engines with 20 membership functions, 7 t-norms, 9
t-conorms, 4 implications, 7 activation methods, 6 defuzzifiers. FCM clustering
with Xie-Beni validation. ANFIS gradient-descent training. Type-2 sets with
Footprint of Uncertainty. SHAP and GNN rule extraction.

**Together.** 3-tier cascade: syntactic check → BDD skeleton equivalence → SAT
solve → tableau proof. Fuzzy bridge resolves continuous-valued atoms into the
Boolean skeleton. PINS dependency matrix for incremental evaluation. Guard-based
partial-order reduction for state-space pruning.

---

## Quick start

```bash
go get github.com/xDarkicex/logic
```

```go
import "github.com/xDarkicex/logic"

// Classical — fluent evaluator
result := logic.Eval(true).And(false).Or(true).Xor(false).Result()

// SAT — solve CNF formulas
solver := logic.NewCDCLSolver()
cnf, _ := logic.ConvertToCNF("(A ∨ B) ∧ (¬A ∨ C) ∧ (¬B ∨ ¬C)")
res := solver.Solve(cnf)
fmt.Println(res.Satisfiable, res.Assignment)

// Fuzzy — Mamdani engine
engine := fuzzy.NewMamdaniEngine(pool)
temp := fuzzy.NewLinguisticVar("temperature")
temp.AddTerm("cold", fuzzy.NewFuzzySet(fuzzy.Triangular(0, 0, 20)))
engine.AddVariable(temp)
output, _ := engine.Evaluate(map[fuzzy.VarID]float64{tempID: 15.0}, pool)
crisp := fuzzy.Centroid(output)
```

---

## Packages

| Package | Purpose | Coverage | Tests |
|---------|---------|----------|-------|
| `classical` | All 7 Boolean gates, fluent evaluator, truth tables, tautology/contradiction, DeMorgan/Distributive, expression parser, bit vectors with bitwise ops, benchmark framework | 71% | 31 |
| `sat` | Kissat-derived CDCL, DPLL fallback, XOR/Gaussian, MAX-SAT, Tseitin CNF, mode switching, WalkSAT, DPLL(T) theory plugins, fuzzy-SMT gradient descent | 77% | ~180 |
| `modal` | Kripke frame, Couvreur emptiness, LTL + epistemic, BDD bridge, hash consing, cascade, bisimulation, world-set, PINS, POR, Zielonka parity, Hopcroft DFA, stutter invariance, LTL splitting, cut-point relabeling, SAT optimizations | 88% | ~390 |
| `fuzzy` | Mamdani + TSK engines, 20 MFs, 7 t-norms, 9 t-conorms, defuzzification, clustering | 89% | ~55 |
| `fuzzy/type2` | Interval Type-2 sets with Footprint of Uncertainty | 99% | — |
| `fuzzy/anfis` | Adaptive Neuro-Fuzzy Inference System, gradient-descent training | 98% | — |
| `fuzzy/xai` | Rule extraction, FCM-based feature importance | 100% | — |
| `core` | Shared interfaces (`LogicSystem`, `EvaluationContext`), error types, engine registry | — | — |

---

## Why xDarkicex/memory

Every backing array in this package — every clause literal slice, every tableau branch, every fuzzy set universe, every SAT solver heap array — is allocated off-heap via [`github.com/xDarkicex/memory`](https://github.com/xDarkicex/memory).

| Allocator | Use in this package |
|-----------|-------------------|
| `memory.Pool` (CAS slab) | Variable-size slices: CNF clauses, fuzzy set universes, parse tokens, VSIDS heap arrays. Bulk `Reset()` between evaluations |
| `memory.Arena` (bump pointer) | Grow-only structures: decision trail entries, tableau branches, frame worlds. `Free()` on teardown |
| `memory.ShardedFreeList` (Hyaline SMR) | Fixed-size high-churn structs: `Clause`, `TableauNode`, `PrefixedFormula`. 25.8× faster than `make()`, zero GC scanning |

There is zero `make([]T, ...)` in this package. The Go GC never traces reasoning data. At scale — 500K tableau nodes, 10K SAT clauses, 64K fuzzy universes — the GC sees nothing.

---

## Why xDarkicex/gobdd

For BDD-based Boolean reasoning, this package integrates with [`github.com/xDarkicex/gobdd`](https://github.com/xDarkicex/gobdd) — the only complete GOBDD library in pure Go with full complement of BDD operations (And, Or, Not, Xor, Implies, Iff, Restrict, Compose, Exist, ForAll, SatisfyOne, SatisfyAll, NodeCount). The modal package's BDD bridge (`modal/bdd_bridge.go`) converts Boolean skeletons to GOBDD canonical forms for O(1) equivalence checking after construction.

For R4, SAT-backed equivalence is sufficient (two SAT calls). BDDs are available for O(1) repeated queries.

---

## Architecture

```
logic.go              Top-level re-exports, backwards compatibility
core/                 Shared interfaces (LogicSystem, EvaluationContext), errors
classical/            Boolean gates, evaluator, parser, truth tables, bit vectors
sat/                  CDCL solver, watched literals, 1st UIP, VSIDS heap, mode switching,
                      WalkSAT, Gaussian elimination, DPLL(T), MAX-SAT, inprocessing
modal/                Kripke frame, tableau prover, temporal/epistemic logic,
                      Couvreur emptiness, BDD bridge, hash consing, cascade,
                      bisimulation, world-set encoding, PINS, POR, Zielonka parity games
fuzzy/                Mamdani/TSK engines, 20 MFs, operators, defuzzification,
                      clustering, ANFIS, Type-2, XAI, hedges, arbiter
fuzzy/type2/          Interval Type-2 fuzzy sets with FOU
fuzzy/anfis/          Neuro-adaptive training via gradient descent
fuzzy/xai/            Explainable AI: rule extraction, feature importance
docs/                 THEORY.md, ARCHITECTURE.md, RESEARCH.md
```

---

## SAT solver

A Kissat-derived CDCL solver. Kissat (Biere, 2020–2024) consistently places top-3 at
the SAT Competition. MIT-licensed. Our solver adapts its architecture to Go with
off-heap memory — same algorithms, zero GC interaction.

| Feature | Algorithm | Impact |
|---------|-----------|--------|
| **Watched literals** | Two-watched-literal scheme (Chaff, 2001) | O(1) amortized propagation |
| **1st UIP analysis** | First Unique Implication Point (Zhang et al., 2001) | Short, reusable learned clauses |
| **LBD tracking** | Literal Block Distance (Glucose, 2009) | Glue clause (LBD ≤ 2) protection |
| **VSIDS heap** | Binary max-heap, O(log n) selection | LRB blending + anti-aging decay |
| **Mode switching** | Focused/stable with reluctant doubling (Kissat) | Exploitation/exploration balance |
| **WalkSAT** | Probabilistic make/break local search | Sub-ms easy solves, warm-start phases |
| **Gaussian elimination** | Gauss-Jordan over GF(2) | Native XOR without exponential CNF blowup |
| **Inprocessing** | BVE, subsumption, vivification (Järvisalo et al., 2012) | Formula reduction between restarts |
| **DPLL(T)** | Theory plugin interface (Z3 architecture) | SMT-style theory integration |
| **MAX-SAT** | Binary search on weight threshold | Weighted maximum satisfiability |

```go
// Full CDCL with XOR support
solver := sat.NewCDCLSolver()
solver.EnableXORSupport(true)

ecnf := sat.NewExtendedCNF()
ecnf.AddClause(sat.NewClause(sat.L("A", false), sat.L("B", false)))
ecnf.AddXORClause(sat.NewXORClause([]string{"A", "B", "C"}, true))

result := solver.SolveExtended(ecnf)
fmt.Println(result.Satisfiable, result.Assignment)
```

---

## Modal logic

Kripke semantics with accessibility relations. Tableau prover with on-the-fly SCC
emptiness (Couvreur, 1999). BDD bridge for O(1) Boolean equivalence after
construction. Hash consing for O(1) structural equality of formulas.

**Axiom systems** — K, D, T, B, S4, S5. Enforce frame properties (reflexivity,
symmetry, transitivity, seriality) on accessibility relations. Validate frames
against axiom systems.

**Temporal logic (LTL)** — □ (always), ◇ (eventually), U (until), ○ (next).
Level-based degeneralization converts generalized Büchi to classical Büchi.
Now/Next/Promise decomposition (Couvreur FM). LTL splitting into obligation /
suspendable / rest components for independent solving.

**Epistemic logic** — Multi-agent Kaφ operators. S5 accessibility for knowledge,
distributed knowledge, and common knowledge. Kripke evaluator over multi-agent
frames.

**State-space reduction** — Bisimulation contraction with BDD-refined signature
matching (Bustan & Grumberg, 2003). Stutter invariance for property-preserving
state merging. Hopcroft partition refinement for DFA minimization. World-set
encoding for O(1) bit-vector world membership.

**Automata and verification** — Zielonka parity game solver for reactive synthesis.
Formula rewriting (simplify, push-negation, NNF). Cut-point relabeling via
articulation points. Independent component decomposition. PINS dependency matrix
for incremental evaluation. Guard-based partial-order reduction. Dead subspace
tracking across CDCL restarts.

**Cascade pipeline** — 3-tier cheap-first: syntactic check → BDD skeleton
equivalence → SAT solve → tableau/Couvreur proof. The BDD bridge converts
modal subformulas to fresh Boolean variables, SAT checks the skeleton, tableau
handles the modal fragment.

```go
ctx := modal.NewBDDCtx(256, pool)
frame := modal.NewFrame(pool, arena)
frame.AddWorld() // w0
frame.AddWorld() // w1
frame.AddAccessibility(modal.RelationK, 0, 1)

// Enforce S5 on epistemic relation
modal.EnforceSystemS5(frame, modal.RelationE)

// Satisfiability via Couvreur on-the-fly emptiness
prover := modal.NewCouvreurProver(pool, arena)
sat, model := prover.ProveSatisfiable(formula, frame)

// Bisimulation contraction
contractor := modal.NewBisimContractor(ctx, pool)
reduced := contractor.Contract(model)

// Cascade: syntactic → BDD → SAT → tableau
result := modal.NewCascade(pool).Prove(formula, frame)
```

---

## Fuzzy logic

A complete fuzzy inference system — Mamdani and TSK engines, 20 membership functions, 7 t-norms, 9 t-conorms, 4 implications, 7 activation methods, 6 defuzzifiers, hedges, linguistic variables with Pool-backed terms, multi-strategy arbitration with reliability tracking.

```go
pool, _ := memory.NewPool(memory.DefaultConfig())
engine := fuzzy.NewMamdaniEngine(pool)

// Build linguistic variable
temp := fuzzy.NewLinguisticVar(fuzzy.VarID(1))
cold := fuzzy.NewFuzzySet(fuzzy.VarID(10), []float64{0, 10, 20, 30, 40}, pool)
cold.Func = fuzzy.Triangular(0, 0, 20)
temp.AddTerm(fuzzy.VarID(10), cold)
engine.AddVariable(temp)

// Add rule with per-block operators
rule := fuzzy.NewFuzzyRule(1.0)
rule.AddAntecedent(fuzzy.VarID(1), fuzzy.VarID(10), false)
rule.SetConsequent(fuzzy.VarID(2), fuzzy.VarID(20))
block := fuzzy.NewRuleBlock("heater", 16, pool)
block.AddRule(*rule)
engine.AddRuleBlock(block)

// Evaluate and defuzzify
result, _ := engine.Evaluate(map[fuzzy.VarID]float64{1: 15.0}, pool)
crisp := fuzzy.Centroid(result)
```

---

## Research backing

This package is grounded in the academic SAT, modal logic, and fuzzy systems literature. Key influences:

**CDCL SAT** — GRASP: A Search Algorithm for Propositional Satisfiability
(Silva & Sakallah, 1999). Chaff: Engineering an Efficient SAT Solver (Moskewicz
et al., 2001). Efficient Conflict Driven Learning (Zhang et al., 2001). Predicting
Learnt Clauses Quality (Audemard & Simon, Glucose 2009). Kissat (Biere, SAT
Competition 2020–2024). WalkSAT (Selman, Kautz, Cohen, AAAI 1994). Inprocessing
Rules (Järvisalo, Heule, Biere, IJCAR 2012).

**Modal and automata** — On-the-fly Emptiness (Couvreur, 1999). Level-Based
Degeneralization (Bloemen et al., 2019). Simulation-Based State Reduction
(Bustan & Grumberg, 2003). Zielonka Parity Games (Zielonka, 1998). Hopcroft
DFA Minimization (Hopcroft, 1971).

**Memory** — Hyaline: Fast and Transparent Lock-Free Memory Reclamation
(Nikolaev & Ravindran, PLDI 2021) via `xDarkicex/memory`.

See [`docs/RESEARCH.md`](docs/RESEARCH.md) for the complete paper list and licensing statement. All algorithms are re-derived from published mathematics. No non-MIT code was used.

---

## When to use

- You need Boolean, SAT, modal, *and* fuzzy reasoning in one dependency
- You are building formal verification, agentic reasoning, or regulatory compliance tools
- GC pressure from reasoning data is a problem — off-heap allocation eliminates it
- You want production-grade SAT solving without linking to C/C++
- You need predictable latency under sustained inference workloads

## When not to use

- You need only one reasoning paradigm and prefer a smaller dependency
- You need SMT theories beyond equality (arithmetic, bit-vectors, arrays) — this is a SAT solver with DPLL(T) plugin support, not a full SMT solver
- You need probabilistic reasoning (Bayesian networks, Markov logic) — this is logical, not probabilistic
- You are on a platform without `mmap` support (Plan 9, NaCl)

---

## Links

- [THEORY.md](docs/THEORY.md) — SAT solving: DPLL through CDCL, watched literals, 1st UIP, VSIDS, LBD, mode switching, WalkSAT, Gaussian elimination, inprocessing, DPLL(T)
- [ARCHITECTURE.md](docs/ARCHITECTURE.md) — package map, off-heap allocation strategy, data flow, component diagram, tuning constants
- [RESEARCH.md](docs/RESEARCH.md) — academic papers, reference implementations, MIT licensing statement
- [xDarkicex/memory](https://github.com/xDarkicex/memory) — off-heap allocator (mmap-backed, lock-free, Hyaline SMR)
- [xDarkicex/gobdd](https://github.com/xDarkicex/gobdd) — GOBDD library in pure Go (BDD bridge for O(1) equivalence)

---

## License

MIT. See [LICENSE](LICENSE).

---

Made with love for Gophers who need rock-solid logic tooling.
