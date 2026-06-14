# Logic — The Logic Engine That Does Everything

[![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/xDarkicex/logic)](https://goreportcard.com/report/github.com/xDarkicex/logic)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/xDarkicex/logic.svg)](https://pkg.go.dev/github.com/xDarkicex/logic)

**The only Go logic library you'll ever need.** From boolean gates to industrial-strength SAT solving to full fuzzy inference — all in one package, all zero-heap by default. No `make([]T, ...)`. No GC pressure. Just raw, alloc-free logical reasoning.

### Why This Exists

Most logic libraries pick one paradigm and stop. This one doesn't. Real systems — agentic AI, formal verification, embedded controllers — need to reason across classical, SAT, and fuzzy domains simultaneously. You shouldn't need three libraries and a memory headache to make that happen.

### What's Under the Hood

| Paradigm | What You Get |
|----------|--------------|
| **Classical** | All 7 gates (AND/OR/XOR/NAND/NOR/NOT/XNOR), implication, biconditional, fluent evaluator, truth tables, tautology/contradiction/contingency detection, DeMorgan/Distributive laws, circuit simulator |
| **SAT** | CDCL solver with First-UIP learning, LBD tracking, VSIDS/LRB/CHB heuristics, XOR reasoning via Gaussian elimination, MaxSAT, inprocessing (BVE, subsumption, probing), Tseitin CNF conversion, DPLL fallback |
| **Fuzzy** | Mamdani + TSK engines, 20 membership functions, 7 t-norms, 9 t-conorms, 4 implications, 7 activation methods, 6 defuzzifiers, `RuleBlock` (per-block operators), `OutputVariable` (per-variable defuzzifier/aggregation), FCM clustering, ANFIS training, Type-2 sets with FOU, hedges, XAI rule extraction, multi-strategy arbiter, numerical utilities (Softplus/Softmax/LogSoftmax/Sparsemax) |
| **Modal** *(in progress)* | Kripke semantics, tableau prover, temporal logic (LTL), epistemic logic, fuzzy-modal bridge |

### Zero-Heap, Low Complexity, Production Grade

- **Off-heap memory.** Every slice, every backing array, every token buffer goes through [`github.com/xDarkicex/memory`](https://github.com/xDarkicex/memory) — Pool for variable-length data, Arena for grow-only structures. The Go GC never scans this package. You get predictable latency under any reasoning workload.
- **Cyclomatic complexity ≤ 10.** Every function — exported or internal — is verified. Hot-path evaluators, tableau expansion, and temporal operators target CC ≤ 6. The codebase stays flat, readable, and auditable.
- **100% test coverage.** `go test -cover ./...` and `go test -race ./...` pass on every phase. Edge cases are not optional.

## Installation

```bash
go get github.com/xDarkicex/logic
```

Minimum Go version: **1.18**.

## Packages

| Package | Purpose |
|---------|---------|
| `logic` | Top-level re-exports, `LogicEngine`, backwards-compatible convenience functions |
| `classical` | Boolean gates, truth tables, expression parser, circuits, tautology/contradiction |
| `sat` | CDCL solver, DPLL, XOR clauses, MaxSAT, CNF conversion (Tseitin), Gaussian elimination, fuzzy-SMT |
| `fuzzy` | Mamdani/TSK engines, membership functions, t-norms/conorms, defuzzification, clustering, ANFIS |
| `fuzzy/type2` | Interval Type-2 fuzzy sets with Footprint of Uncertainty |
| `fuzzy/anfis` | Adaptive Neuro-Fuzzy Inference System with gradient-descent training |
| `fuzzy/xai` | Explainable AI: rule extraction and FCM-based explanation |
| `core` | Shared interfaces (`LogicSystem`, `EvaluationContext`), error types, engine registry |
| `modal` | Planned — Kripke semantics, tableau prover, temporal/epistemic logic |

## Quick Start

### Classical Boolean Logic

```go
import "github.com/xDarkicex/logic"

// Variadic gates
logic.And(true, false, true)  // false
logic.Xor(true, false)        // true
logic.Xnor(true, true)        // true
logic.Implies(true, false)    // false
logic.Iff(true, true)         // true

// Fluent chains
out := logic.Eval(true).And(false).Or(true).Result() // true

// Expression parsing
val, _ := logic.EvaluateExpression("A ∧ ¬B → C", map[string]bool{"A": true, "B": false})

// Truth tables — auto-generate for any n-ary function
table := logic.GenerateTruthTable([]string{"P", "Q"}, func(p, q bool) bool {
    return logic.Implies(p, q)
})
fmt.Print(table.String())

// Tautology / Contradiction / Contingency
logic.Tautology(func(p, q bool) bool { return logic.Or(p, logic.Not(p)) })   // true
logic.Contradiction(func(p, q bool) bool { return logic.And(p, logic.Not(p)) }) // true

// DeMorgan / Distributive law verification
logic.DeMorganLaw(true, false)
logic.DistributiveLaw(true, false, true)
```

### SAT Solver (CDCL)

```go
// Quick solve
res, _ := logic.SolveSAT("(A ∨ B) ∧ (¬A ∨ C) ∧ (¬B ∨ ¬C)")
fmt.Println(res.Satisfiable) // true
fmt.Println(res.Model)       // map[A:true B:false C:true]

// Full control
solver := sat.NewCDCLSolver()
solver.EnableXORSupport(true)

cnf := sat.NewCNF()
cnf.AddClause(sat.C("A", "B"))       // A ∨ B
cnf.AddClause(sat.C("¬A", "C"))      // ¬A ∨ C
cnf.AddXORClause(sat.NewXORClause([]string{"A", "B", "C"}, true))

res = solver.Solve(cnf)
res = solver.SolveWithTimeout(cnf, 5*time.Second)

// MaxSAT
maxRes := sat.MaxSATSolve(cnf)
```

#### SAT Features

| Feature | Description |
|---------|-------------|
| **CDCL** | Conflict-Driven Clause Learning with First-UIP analysis and LBD tracking |
| **DPLL** | Classic Davis-Putnam-Logemann-Loveland with unit propagation and pure literal elimination |
| **XOR reasoning** | Gaussian elimination for parity constraints |
| **Inprocessor** | Bounded variable elimination, subsumption, self-subsumption, failed literal probing |
| **Preprocessor** | Pure literal elimination, unit propagation, tautology removal |
| **Heuristics** | VSIDS, VMTF, LRB, CHB, random, static — unified interface |
| **Trail** | Decision trail with level tracking, propagation graphs, reason clauses |
| **MaxSAT** | Unweighted MaxSAT via cardinality constraints |
| **Fuzzy-SMT** | Gradient-descent solving of fuzzy-weighted SAT clauses |
| **CNF Converter** | Tseitin transformation for arbitrary boolean expressions → CNF |

### Fuzzy Logic System

```go
import (
    "github.com/xDarkicex/logic/fuzzy"
    "github.com/xDarkicex/memory"
)

pool := memory.NewPool()
arena := memory.NewArena()

// Create linguistic variable with membership functions
temp := fuzzy.NewLinguisticVar(fuzzy.VarID(1))
cold := fuzzy.NewFuzzySet(fuzzy.VarID(10), []float64{0, 10, 20, 30, 40}, pool)
hot  := fuzzy.NewFuzzySet(fuzzy.VarID(11), []float64{0, 10, 20, 30, 40}, pool)

cold.Func = fuzzy.Triangular(0, 0, 20)
hot.Func  = fuzzy.Triangular(30, 40, 40)

temp.AddTerm(fuzzy.VarID(10), cold)
temp.AddTerm(fuzzy.VarID(11), hot)

// Build rules
rule := fuzzy.NewFuzzyRule(1.0)
rule.AddAntecedent(fuzzy.VarID(1), fuzzy.VarID(10), false) // IF temp IS cold
rule.SetConsequent(fuzzy.VarID(2), fuzzy.VarID(20))        // THEN heater IS high

// Mamdani engine
engine := fuzzy.NewMamdaniEngine(pool)
engine.AddVariable(temp)
engine.AddRule(*rule)

// Evaluate and defuzzify
result, _ := engine.Evaluate(map[fuzzy.VarID]float64{fuzzy.VarID(1): 5.0}, pool)
crisp := fuzzy.Centroid(result)

// TSK engine (Sugeno — function approximation)
tsk := fuzzy.NewTSKEngine()
tsk.AddVariable(temp)
tsk.AddRule(fuzzy.TSKRule{
    Antecedents: []fuzzy.FuzzyCondition{
        {Variable: fuzzy.VarID(1), Term: fuzzy.VarID(10)},
    },
    Consequent: fuzzy.LinearTSK{
        Coeffs:    map[fuzzy.VarID]float64{fuzzy.VarID(1): 0.8},
        Intercept: 2.0,
    },
})
output, _ := tsk.Evaluate(inputs)
```

#### Membership Functions (20 total)

```go
// Basic
fuzzy.Triangular(a, b, c float64)      // Triangle: peak at b, support [a, c]
fuzzy.Trapezoidal(a, b, c, d float64)   // Trapezoid: flat top [b, c]
fuzzy.Rectangle(start, end float64)     // Rectangle: 1 inside [start, end], 0 outside
fuzzy.Singleton(value float64)           // Single crisp value
fuzzy.Discrete(xyPairs []float64, pool) // Table-based (x,y pairs) with linear interpolation

// Curves
fuzzy.Gaussian(mean, stddev float64)     // Gaussian bell curve
fuzzy.GaussianProduct(mean, σLeft, σRight) // Asymmetric Gaussian
fuzzy.Bell(a, b, c float64)             // Generalized bell: 1/(1 + |(x-c)/a|^2b)
fuzzy.Laplace(loc, scale float64)        // Laplace distribution
fuzzy.Cosine(center, width float64)      // Cosine-based smooth MF

// Sigmoid family
fuzzy.Sigmoid(a, c float64)             // Sigmoid: 1/(1 + exp(-a*(x-c)))
fuzzy.SigmoidDifference(l, li, r, ri)   // Difference of two sigmoids (bump)
fuzzy.SigmoidProduct(l, li, r, ri)      // Product of two sigmoids (smooth bump)

// Monotonic
fuzzy.Ramp(start, end float64)          // Linear ramp: 0→1
fuzzy.SShape(start, end float64)        // S-shaped growth: quadratic spline
fuzzy.ZShape(start, end float64)        // Z-shaped decay: quadratic spline
fuzzy.Concave(start, end float64)       // Concave rise: diminishing returns
fuzzy.Binary(inflection, direction)     // Threshold: 0/1 based on direction

// Composite
fuzzy.PiShape(bottom, top, start, end)  // Pi-shaped bump: SShape × ZShape
fuzzy.Spike(center, width float64)      // Spike: exp(-|x-center|/width)
```

#### Operators

**T-Norms (Fuzzy AND — 7 total):** `MinTNorm`, `ProductTNorm`, `LukasiewiczTNorm`, `EinsteinProduct`, `HamacherProduct`, `NilpotentMinimum`, `DrasticProduct`, `MinTNormVariadic`

**T-Conorms (Fuzzy OR — 9 total):** `MaxTConorm`, `ProbabilisticTConorm`, `LukasiewiczTConorm`, `EinsteinSum`, `HamacherSum`, `NilpotentMaximum`, `DrasticSum`, `NormalizedSum`, `UnboundedSum`, `MaxTConormVariadic`

**Implications:** `GodelImplication`, `GoguenImplication`, `LukasiewiczImplication`, `KleeneDienesImplication`

**Negation:** `StandardNegation`

**Hedges:** `Very`, `Somewhat`, `Slightly`, `Extremely`, `Indeed`, `Not`

**Activation Methods (7 total):** `General`, `Proportional` (softmax-normalize), `Threshold` (6 comparison operators), `First`, `Last`, `Highest`, `Lowest` — pluggable per `RuleBlock`

**Numerical Utilities (from gorgonia):** `Softplus` (stable `log(1+e^x)`), `Softmax`, `LogSoftmax`, `Sparsemax` (sparse probability projection)

#### Defuzzification

| Method | Function |
|--------|----------|
| Centroid (center of area) | `fuzzy.Centroid(set)` |
| Mean of Max | `fuzzy.MeanOfMax(set)` |
| Smallest of Max | `fuzzy.SmallestOfMax(set)` |
| Largest of Max | `fuzzy.LargestOfMax(set)` |
| Bisector | `fuzzy.Bisector(set)` |
| Weighted Average | `fuzzy.WeightedAverageDefuzz(values, weights)` |

### Type-2 Fuzzy Logic

```go
import "github.com/xDarkicex/logic/fuzzy/type2"

upper := fuzzy.NewFuzzySet(id, universe, pool)
lower := fuzzy.NewFuzzySet(id, universe, pool)
upper.Func = fuzzy.Gaussian(0, 1.0)
lower.Func = fuzzy.Gaussian(0, 0.5)

it2set, _ := type2.NewIntervalType2Set(id, upper, lower)
fou := it2set.Membership(0.5) // FOUInterval{Lower: ..., Upper: ...}

it2var := type2.NewType2LinguisticVar(id)
it2var.AddTerm(termID, it2set)

engine := type2.NewType2MamdaniEngine(pool)
engine.AddVariable(it2var)
engine.AddRule(rule)
result, _ := engine.Evaluate(inputs, pool)
```

### ANFIS (Adaptive Neuro-Fuzzy)

```go
import "github.com/xDarkicex/logic/fuzzy/anfis"

anfis := anfis.NewANFIS(pool)
// Add Gaussian MFs and TSK rules, then train:
anfis.Train(epochs=100, learningRate=0.01, inputs, targets)
```

### Fuzzy Clustering (FCM)

```go
data := [][]float64{{1.0, 2.0}, {1.5, 1.8}, {5.0, 8.0}, {8.0, 8.0}}
centroids, weights, _ := fuzzy.Cluster(data, fuzzy.FCMConfig{
    Clusters: 2, Fuzziness: 2.0, MaxIters: 100, Epsilon: 1e-5,
})
memberships := fuzzy.PredictMembership([]float64{2.0, 3.0}, centroids, 2.0, cfg)
fuzzy.FPC(weights, clusters, samples)              // Fuzzy Partition Coefficient
fuzzy.XieBeni(data, centroids, weights, cfg)        // Xie-Beni index
optimalC := fuzzy.OptimizeClusterCount(data, maxC, cfg)
```

### XAI (Explainable AI)

```go
import "github.com/xDarkicex/logic/fuzzy/xai"

rules := xai.ExtractRules(engine, threshold=0.5)
importance := xai.FCMFeatureImportance(data, centroids, weights)
```

### Multi-Strategy Arbiter

```go
arbiter := fuzzy.NewArbiter()
arbiter.AddStrategy("mamdani", &fuzzy.MamdaniStrategy{Engine: mEngine}, 0.8)
arbiter.AddStrategy("tsk", &fuzzy.TSKStrategy{Engine: tEngine}, 0.7)
result, strategy, _ := arbiter.Select(inputs, pool)
arbiter.UpdateReliability("tsk", +0.05)
```

### RuleBlock & OutputVariable (port from fuzzylite)

```go
// Each RuleBlock has its own conjunction, disjunction, implication, and activation
block := fuzzy.NewRuleBlock("safety-rules", 16, pool)
block.SetConjunction(fuzzy.MinTNorm)
block.SetImplication(fuzzy.GodelImplication)
block.SetActivation(fuzzy.NewThresholdActivation(0.3, fuzzy.GreaterThan))
block.AddRule(*rule)

// OutputVariables have per-variable defuzzification, aggregation, and defaults
ov := fuzzy.NewOutputVariable(linguisticVar)
ov.SetDefuzzifier(fuzzy.NewCentroidDefuzzifier())
ov.SetAggregation(fuzzy.MaxTConorm)
ov.SetDefaultValue(25.0)
ov.SetLockValidRange(0, 100)

engine.AddRuleBlock(block)
engine.AddOutputVariable(ov)
```

## Architecture

```
logic.go              → top-level re-exports, backwards compatibility
core/
  interfaces.go       → LogicSystem, EvaluationContext, Engine
  types.go            → LogicEngine registry
  errors.go           → LogicError
classical/
  gates.go            → AND, OR, XOR, XNOR, NAND, NOR, NOT, Implies, Iff
  evaluator.go        → Fluent expression evaluator
  parser.go           → Expression lexer/parser (∧, ∨, ¬, →, ↔)
  truthtable.go       → Truth table generator, tautology/contradiction
  bitvector.go        → BoolVector with variadic ops
  logic.go            → BitwiseInt wrapper, DeMorgan, Distributive
  system.go           → core.LogicSystem adapter
sat/
  types.go            → CNF, Clause, Literal, Assignment, SolverResult
  cdcl.go             → CDCL solver with VSIDS, LBD, restarts
  dpll.go             → DPLL solver
  conflict_analysis.go → First-UIP clause learning
  heuristics.go       → VSIDS, VMTF, LRB, CHB decision heuristics
  trail.go            → Decision trail with level/propagation tracking
  inprocessor.go      → Inprocessing: BVE, subsumption, probing
  preprocessor.go     → Preprocessing: pure literal, tautology removal
  cnf_converter.go    → Tseitin transformation
  gaussian.go         → Gaussian elimination for XOR clauses
  maxsat.go           → MaxSAT solver
  fuzzy_smt.go        → Fuzzy-SMT gradient-descent solver
  system.go           → core.LogicSystem adapter
fuzzy/
  types.go            → TruthValue, VarID, SymbolTable, FuzzySet, LinguisticVar (Pool-backed terms), FuzzyRule (OR antecedents)
  membership.go       → 20 MFs: Triangular, Trapezoidal, Rectangle, Gaussian, GaussianProduct, Bell, Laplace, Cosine, Sigmoid, SigmoidDifference, SigmoidProduct, Singleton, Discrete, Ramp, SShape, ZShape, Concave, Binary, PiShape, Spike
  operators.go        → 7 t-norms, 9 t-conorms, 4 implications, negation
  activation.go       → 7 methods: General, Proportional, Threshold (6 ops), First, Last, Highest, Lowest
  inference.go        → MamdaniEngine (RuleBlock + OutputVariable), TSKEngine (activation + OR antecedents)
  ruleblock.go        → RuleBlock: per-block conjunction/disjunction/implication/activation + Pool-backed rules
  variable.go         → OutputVariable (per-variable defuzzifier/aggregation/default/lock-range), Defuzzifier interface (5 wrappers)
  defuzzify.go        → Centroid, MeanOfMax, SmallestOfMax, LargestOfMax, Bisector, WeightedAverage
  math.go             → Softplus, Softmax, LogSoftmax (from gorgonia)
  sparsemax.go        → Sparsemax: sparse probability projection (from gorgonia)
  parser.go           → Fuzzy rule lexer/parser (Pool-backed tokens)
  sets.go             → Union, Intersection, Complement, Concentration, Dilation, CartesianProduct
  hedges.go           → Very, Somewhat, Slightly, Extremely, Indeed, Not
  cluster.go          → FCM clustering, Xie-Beni, FPC, optimal cluster count
  encoding.go         → Population encoder (Pool-backed)
  arbiter.go          → Multi-strategy selection with reliability tracking
  system.go           → core.LogicSystem adapter
fuzzy/type2/
  types.go            → IntervalType2Set, FOUInterval, Type2LinguisticVar
  inference.go        → Type-2 Mamdani engine with type reduction
fuzzy/anfis/
  anfis.go            → ANFIS with gradient-descent training
fuzzy/xai/
  extract.go          → Rule extraction from trained systems
  fcm.go              → FCM-based feature importance
```

## Testing

```bash
go test ./...                    # all packages
go test -cover ./...             # with coverage
go test -race ./...              # race detector
go test -bench=. -benchmem ./... # benchmarks
```

## Error Handling

All errors are `*core.LogicError` with module, operation, and message:

```go
_, err := logic.EvaluateExpression("A ∧", vars)
// &LogicError{Module: "classical", Operation: "EvaluateExpression", Message: "..."}
```

## Contributing

1. Fork the repo
2. Create a feature branch (`git checkout -b feat/xor-heuristic`)
3. Add tests and benchmarks
4. Run `gofmt`, `golangci-lint`, and `go test ./...`
5. Open a PR with a clear description

See [CONTRIBUTING.md](CONTRIBUTING.md) for coding standards.

## License

MIT. See [LICENSE](LICENSE) for details.

---

Made with ❤️ for Go developers who need rock-solid logic tooling.  
⭐ Star the repo if this helped you!
