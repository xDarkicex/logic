# logic

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/xDarkicex/logic)](https://goreportcard.com/report/github.com/xDarkicex/logic)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/xDarkicex/logic.svg)](https://pkg.go.dev/github.com/xDarkicex/logic)
[![Tests](https://img.shields.io/badge/tests-670+-blue.svg)]()
[![Coverage](https://img.shields.io/badge/coverage-77--100%25-brightgreen.svg)]()
[![Race](https://img.shields.io/badge/race-clean-brightgreen.svg)]()

**A complete logic reasoning engine in pure Go.** Classical propositional logic, industrial-strength CDCL SAT solving (Kissat-derived), modal logic with Kripke semantics and temporal/epistemic reasoning, and full fuzzy inference — all off-heap, all zero `make([]T)`, all race-clean.

670+ tests. 77–100% coverage across 8 packages. The Go GC never scans reasoning data.

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

**Classical:** `(A → B) ∧ (B → C) → (A → C)` — is this a tautology? Instant answer with truth tables or the fluent evaluator.

**SAT:** Is there an assignment satisfying 10,000 Boolean constraints with XOR parity requirements? CDCL with Gaussian elimination finds it or proves unsatisfiability.

**Modal:** Does agent *a* know that agent *b* knows φ in an S5 epistemic frame? Kripke evaluator with multi-agent accessibility relations.

**Temporal:** Does this LTL property hold along all session timelines? Tableau prover with Couvreur on-the-fly SCC emptiness.

**Fuzzy:** Given temperature = 22°C and humidity = 65%, what heater power is recommended? Mamdani engine with 20 membership functions and centroid defuzzification.

**Together:** Is the agent's belief state logically consistent with the temporal specification, under fuzzy uncertainty? Cascade: SAT checks the Boolean skeleton, tableau checks the modal fragment, fuzzy bridge resolves continuous-valued atoms.

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
| `classical` | Boolean gates, truth tables, tautology/contradiction, expression parser, bit vectors | 71% | 31 |
| `sat` | CDCL solver, DPLL, XOR/Gaussian, MAX-SAT, Tseitin CNF, mode switching, WalkSAT, DPLL(T) | 77% | ~180 |
| `modal` | Kripke semantics, tableau prover, LTL, epistemic logic, axioms, BDD bridge, hash consing, cascade, bisimulation, POR | 88% | ~390 |
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

For BDD-based Boolean reasoning, this package integrates with [`github.com/xDarkicex/gobdd`](https://github.com/xDarkicex/gobdd) — the only complete ROBDD library in pure Go with full complement of BDD operations (And, Or, Not, Xor, Implies, Iff, Restrict, Compose, Exist, ForAll, SatisfyOne, SatisfyAll, NodeCount). The modal package's BDD bridge (`modal/bdd_bridge.go`) converts Boolean skeletons to GOBDD canonical forms for O(1) equivalence checking after construction.

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

## SAT Solver

A Kissat-derived CDCL solver with modern optimizations.

| Feature | Description |
|---------|-------------|
| **Watched literals** | Two-watched-literal scheme, O(1) amortized propagation |
| **1st UIP analysis** | First Unique Implication Point with LBD-based clause quality |
| **VSIDS heap** | Binary max-heap, O(log n) variable selection with LRB + anti-aging |
| **Mode switching** | Focused/stable alternation with reluctant doubling (Kissat-derived) |
| **WalkSAT** | Pre-solving via probabilistic local search, phase warm-start for CDCL |
| **Gaussian elimination** | Native XOR constraint handling over GF(2) |
| **Inprocessing** | Bounded variable elimination, subsumption, failed literal probing |
| **DPLL(T)** | Theory solver integration (Z3-derived architecture) |
| **MAX-SAT** | Weighted maximum satisfiability via binary search |

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

## Fuzzy Logic

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

**CDCL SAT**: GRASP (Silva & Sakallah 1999), Chaff (Moskewicz et al. 2001), 1st UIP (Zhang et al. 2001), Glucose/LBD (Audemard & Simon 2009), Kissat (Biere 2020–2024)

**Modal logic**: On-the-fly emptiness (Couvreur 1999), Level-based degeneralization (Bloemen et al. 2019), Simulation-based reduction (Bustan & Grumberg 2003), Zielonka parity games (Zielonka 1998)

**Memory management**: Hyaline SMR (Nikolaev & Ravindran, PLDI 2021) via `xDarkicex/memory`

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
- [xDarkicex/gobdd](https://github.com/xDarkicex/gobdd) — ROBDD library in pure Go (BDD bridge for O(1) equivalence)

---

## License

MIT. See [LICENSE](LICENSE).
