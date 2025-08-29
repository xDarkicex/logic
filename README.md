# Logic – Advanced Logic & SAT Library for Go

[![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/xDarkicex/logic)](https://goreportcard.com/report/github.com/xDarkicex/logic)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/xDarkicex/logic.svg)](https://pkg.go.dev/github.com/xDarkicex/logic)

A production-ready Go library that combines **high-performance boolean logic**, **bitwise manipulation**, **truth-table generation**, and a **modern CDCL SAT solver**. Designed for digital-logic education, formal verification, AI constraint solving, and embedded systems.

---

## Highlights

| Feature | What You Get |
|---------|--------------|
| **Boolean Engine** | All gates (AND, OR, XOR, NAND, NOR, NOT, XNOR), implications, biconditionals |
| **Bitwise Toolkit** | Individual-bit ops, population count, power-of-2, SIMD-friendly helpers |
| **Truth Tables** | Auto-generation for any n-ary function, tautology/contradiction detection |
| **SAT Solver** | Conflict-Driven Clause Learning (CDCL), XOR clauses, timeout, statistics |
| **Circuit Builder** | Declarative netlists with dependency resolution |
| **Fluent API** | Chainable expressions for human-readable code |

---

## Installation

```bash
go get github.com/xDarkicex/logic
```

Minimum Go version: **1.18** (generics, `math/bits`).

---

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/xDarkicex/logic"
)

func main() {
    // 1. Basic gates
    fmt.Println(logic.And(true, false, true)) // false

    // 2. Fluent chains
    out := logic.Eval(true).And(false).Or(true).Result()
    fmt.Println(out) // true

    // 3. Bitwise on 64-bit ints
    a := logic.NewBitwiseInt(0b1010)
    b := logic.NewBitwiseInt(0b1100)
    fmt.Printf("AND: %04b\n", a.And(b).Value()) // 1000

    // 4. SAT example
    solver := logic.NewCDCLSolver()
    cnf := logic.MustCNF("(A ∨ B) ∧ (¬A ∨ C)")
    res := solver.Solve(cnf)
    fmt.Println("Satisfiable:", res.Satisfiable)
}
```

---

## API Overview

### Core Boolean Operations (variadic)

```go
logic.And(a, b, c, ...)
logic.Or(a, b, c, ...)
logic.Xor(a, b, c, ...)
logic.Nand(a, b, c, ...)
logic.Nor(a, b, c, ...)
logic.Not(a)
logic.Implies(a, b)   // a → b
logic.Iff(a, b)       // a ↔ b
```

### Boolean Vectors

```go
v1 := logic.NewBoolVector(true, false, true)
v2 := logic.NewBoolVector(false, true, true)

and, _ := v1.And(v2) // [false, false, true]
or, _  := v1.Or(v2)  // [true, true, true]
not    := v1.Not()   // [false, true, false]

fmt.Println(and.Count())     // 1
fmt.Println(and.AnyTrue())   // true
fmt.Println(and.AllTrue())   // false
```

### Bitwise Integer Wrapper

```go
n := logic.NewBitwiseInt(42) // 0b101010
n = n.SetBit(0).ToggleBit(5).ClearBit(1)
fmt.Println(n.GetBit(0))      // true
fmt.Println(n.CountSetBits()) // 3
fmt.Println(n.IsPowerOfTwo()) // false
```

### Truth-Table Generator

```go
variables := []string{"P", "Q"}
expr := func(p, q bool) bool { return logic.Implies(p, q) }

table := logic.GenerateTruthTable(variables, expr)
fmt.Print(table.String())
/*
P       Q       Output
----------------------
false   false   true
false   true    true
true    false   false
true    true    true
*/
```

### Circuit Builder

```go
c := logic.NewCircuit([]string{"A", "B"})
c.AddGate("and1", logic.AndGate{}, []string{"A", "B"})
c.AddGate("inv",  logic.NotGate{}, []string{"and1"})
c.SetOutputs([]string{"inv"})

out, _ := c.Simulate(map[string]bool{"A": true, "B": false})
fmt.Println(out["inv"]) // true
```

### CDCL SAT Solver

```go
solver := logic.NewCDCLSolver()
cnf := logic.NewCNF()
cnf.AddClause(logic.C("A", "B"))        // A ∨ B
cnf.AddClause(logic.C("¬A", "C"))       // ¬A ∨ C
cnf.AddClause(logic.C("¬B", "¬C"))      // ¬B ∨ ¬C

res := solver.Solve(cnf)
fmt.Println("Satisfiable:", res.Satisfiable)
fmt.Println("Model:", res.Model)        // map[string]bool
```

---

## Advanced Usage

### Parsing Expressions

```go
vars := map[string]bool{"A": true, "B": false}
val, err := logic.EvaluateExpression("A ∧ ¬B → C", vars)
```

### XOR Support

```go
solver.EnableXORSupport(true)
ecnf := logic.NewExtendedCNF()
ecnf.AddXORClause(logic.NewXORClause([]string{"A", "B", "C"}, true))
```

### Timeout & Statistics

```go
res := solver.SolveWithTimeout(cnf, 5*time.Second)
fmt.Println("Conflicts:", res.Statistics.Conflicts)
```

---

## Error Handling

All vector and circuit functions return descriptive `*logic.LogicError` values:

```go
v1 := logic.NewBoolVector(true, false)
v2 := logic.NewBoolVector(true)

_, err := v1.And(v2)
if err != nil {
    fmt.Println(err) // "vector length mismatch: 2 vs 1"
}
```

---

## Testing & Benchmarks

```bash
# Run tests
go test ./...

# Coverage
go test -cover ./...

# Benchmarks
go test -bench=. -benchmem
```

---

## Roadmap

- [ ] **Incremental SAT API** for dynamic clause addition  
- [ ] **DRAT proof logging** for certified unsatisfiability  
- [ ] **Fuzzy module** for fuzzy logic
- [ ] **Modal logic** AI/LLM logic

---

## Contributing

1. Fork the repo  
2. Create a feature branch (`git checkout -b feat/xor-heuristic`)  
3. Add tests and benchmarks  
4. Run `gofmt`, `golangci-lint`, and `go test ./...`  
5. Open a PR with a clear description  

See [CONTRIBUTING.md](CONTRIBUTING.md) for coding standards.

---

## License

MIT. See [LICENSE](LICENSE) for details.

---

Made with ❤️ for Go developers who need rock-solid logic tooling.  
⭐ Star the repo if this helped you!