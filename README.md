# Logic - Advanced Logic Operations for Go

[![Go Version](https://img.shields.io/badge/Go-1.18+-blue.svg)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/xDarkicex/logic)](https://goreportcard.com/report/github.com/xDarkicex/logic)
[![Documentation](https://pkg.go.dev/badge/github.com/xDarkicex/logic.svg)](https://pkg.go.dev/github.com/xDarkicex/logic)

A comprehensive Go library for logical operations, bitwise manipulation, truth tables, and circuit simulation. Perfect for educational purposes, algorithm development, and digital logic design.

## Features

### 🔧 Core Boolean Operations
- **Basic Logic Gates**: AND, OR, XOR, NOT, NAND, NOR
- **Advanced Operations**: Implies (→), Iff (↔) Biconditional
- **Variadic Support**: Operations on multiple inputs
- **Fluent Interface**: Chain operations with method chaining

### 🧮 Boolean Vector Operations
- Element-wise operations on boolean arrays
- Statistical functions (count, any, all)
- Vector comparisons and manipulations
- String representations for debugging

### ⚡ Bitwise Operations
- Complete bitwise operation suite
- Individual bit manipulation (set, clear, toggle, get)
- Bit shifting operations
- Population count and power-of-2 detection
- Format conversions

### 📊 Truth Table Generation
- Automatic truth table generation for any logical function
- Formatted output with proper alignment
- Support for arbitrary number of variables
- Tautology and contradiction detection

### 🔌 Circuit Simulation
- Logic gate implementations
- Simple circuit construction and simulation
- Extensible gate interface for custom gates

### 🧪 Logical Law Verification
- Built-in implementations of De Morgan's Laws
- Distributive law verification
- Custom logical law testing framework

## Installation

```bash
go get github.com/yourusername/logic
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/yourusername/logic"
)

func main() {
    // Basic operations
    result := logic.And(true, false, true) // false
    fmt.Printf("AND result: %v\n", result)
    
    // Fluent interface
    chainResult := logic.Eval(true).And(false).Or(true).Result() // true
    fmt.Printf("Chain result: %v\n", chainResult)
    
    // Bitwise operations
    a := logic.NewBitwiseInt(0b1010)
    b := logic.NewBitwiseInt(0b1100)
    fmt.Printf("Bitwise AND: %b\n", a.And(b).Value()) // 1000
    
    // Boolean vectors
    v1 := logic.NewBoolVector(true, false, true)
    v2 := logic.NewBoolVector(false, true, true)
    andResult, _ := v1.And(v2)
    fmt.Printf("Vector AND: %s\n", andResult.String()) // [F, F, T]
}
```

## API Reference

### Basic Boolean Operations

#### Variadic Functions
```go
func And(values ...bool) bool
func Or(values ...bool) bool  
func Xor(values ...bool) bool
func Nand(values ...bool) bool
func Nor(values ...bool) bool
```

#### Binary Operations
```go
func Not(value bool) bool
func Implies(a, b bool) bool  // A → B
func Iff(a, b bool) bool      // A ↔ B (biconditional)
```

### Boolean Vector Operations

```go
type BoolVector []bool

// Create vector
func NewBoolVector(values ...bool) BoolVector

// Operations
func (bv BoolVector) And(other BoolVector) (BoolVector, error)
func (bv BoolVector) Or(other BoolVector) (BoolVector, error)
func (bv BoolVector) Xor(other BoolVector) (BoolVector, error)
func (bv BoolVector) Not() BoolVector

// Statistics
func (bv BoolVector) Count() int        // Count of true values
func (bv BoolVector) AllTrue() bool     // All elements true
func (bv BoolVector) AnyTrue() bool     // Any element true
func (bv BoolVector) String() string    // Formatted string
```

### Bitwise Operations

```go
type BitwiseInt struct {
    // Wraps uint64 for enhanced bitwise operations
}

// Create
func NewBitwiseInt(value uint64) BitwiseInt

// Basic operations
func (bi BitwiseInt) And(other BitwiseInt) BitwiseInt
func (bi BitwiseInt) Or(other BitwiseInt) BitwiseInt
func (bi BitwiseInt) Xor(other BitwiseInt) BitwiseInt
func (bi BitwiseInt) Not() BitwiseInt

// Bit manipulation
func (bi BitwiseInt) SetBit(position uint) BitwiseInt
func (bi BitwiseInt) ClearBit(position uint) BitwiseInt
func (bi BitwiseInt) ToggleBit(position uint) BitwiseInt
func (bi BitwiseInt) GetBit(position uint) bool

// Shifting
func (bi BitwiseInt) LeftShift(positions uint) BitwiseInt
func (bi BitwiseInt) RightShift(positions uint) BitwiseInt

// Queries
func (bi BitwiseInt) CountSetBits() int
func (bi BitwiseInt) IsPowerOfTwo() bool
func (bi BitwiseInt) Value() uint64
func (bi BitwiseInt) ToBoolVector() BoolVector
```

### Truth Tables

```go
type TruthTable struct {
    Variables []string
    Rows      []TruthTableRow
}

type TruthTableRow struct {
    Inputs map[string]bool
    Output bool
}

// Generate truth table
func GenerateTruthTable(variables []string, fn func(...bool) bool) *TruthTable

// Analyze logical functions
func Tautology(variables []string, fn func(...bool) bool) bool
func Contradiction(variables []string, fn func(...bool) bool) bool
```

### Fluent Interface

```go
// Chain operations fluently
func Eval(value bool) *Evaluator
func (e *Evaluator) And(other bool) *Evaluator
func (e *Evaluator) Or(other bool) *Evaluator
func (e *Evaluator) Xor(other bool) *Evaluator
func (e *Evaluator) Not() *Evaluator
func (e *Evaluator) Result() bool
```

### Logic Gates & Circuits

```go
type Gate interface {
    Evaluate(inputs ...bool) bool
    String() string
}

// Available gates: AndGate, OrGate, NotGate, XorGate, NandGate, NorGate

type Circuit struct{}

func NewCircuit(inputs []string) *Circuit
func (c *Circuit) AddGate(gate Gate)
func (c *Circuit) Simulate(inputs map[string]bool) (bool, error)
```

## Examples

### Truth Table Generation

```go
// Generate truth table for XOR
table := logic.GenerateTruthTable(
    []string{"A", "B"}, 
    func(inputs ...bool) bool {
        return logic.Xor(inputs...)
    },
)

fmt.Print(table.String())
// Output:
// A       B       Output
// ------------------
// F       F       F
// F       T       T  
// T       F       T
// T       T       F
```

### Logical Law Verification

```go
// Verify De Morgan's Law: !(A ∧ B) = !A ∨ !B
variables := []string{"A", "B"}
deMorganLaw := func(inputs ...bool) bool {
    return logic.DeMorganLaw(inputs[0], inputs[1])
}

isTautology := logic.Tautology(variables, deMorganLaw)
fmt.Printf("De Morgan's Law is valid: %v\n", isTautology) // true
```

### Advanced Bitwise Operations

```go
num := logic.NewBitwiseInt(42) // Binary: 101010

// Bit manipulation
modified := num.SetBit(0).ClearBit(2).ToggleBit(5)

// Analysis
popCount := num.CountSetBits()          // 3
isPowerOf2 := num.IsPowerOfTwo()        // false
rightmostBit := num.GetBit(1)           // true

// Convert to boolean vector
boolVec := num.ToBoolVector()
fmt.Printf("First 8 bits: %v\n", boolVec[:8])
```

### Boolean Vector Operations

```go
v1 := logic.NewBoolVector(true, false, true, false)
v2 := logic.NewBoolVector(false, true, true, false)

// Element-wise operations
and, _ := v1.And(v2)    // [F, F, T, F]
or, _ := v1.Or(v2)      // [T, T, T, F]
xor, _ := v1.Xor(v2)    // [T, T, F, F]
not := v1.Not()         // [F, T, F, T]

// Statistics
fmt.Printf("V1 has %d true values\n", v1.Count())
fmt.Printf("All true: %v, Any true: %v\n", v1.AllTrue(), v1.AnyTrue())
```

## Performance

The package is optimized for both correctness and performance:

- **Basic Operations**: Inline-friendly implementations
- **Bitwise Operations**: Direct hardware instruction mapping
- **Vector Operations**: Loop unrolling for small vectors
- **Truth Tables**: Optimized generation algorithms

Run benchmarks:
```bash
go test -bench=.
```

## Testing

Comprehensive test suite with full coverage:

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test categories
go test -run TestBasicOperations
go test -run TestBitwiseOperations  
go test -run TestBoolVector
```

## Error Handling

The package uses structured error handling:

```go
type LogicError struct {
    Op      string  // Operation name
    Message string  // Error message
}
```

Common errors:
- Vector length mismatches in boolean vector operations
- Invalid circuit inputs (circuits with no gates)
- Empty operation inputs where not supported

## Current Limitations

- **Circuit Simulation**: Current implementation only evaluates the first gate in a circuit
- **XNOR Operation**: Not directly implemented (can be created as `Not(Xor(...))`)
- **Contingency Detection**: Not yet implemented

## Contributing

We welcome contributions! Here's how to get started:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Add tests for new functionality
4. Commit your changes (`git commit -am 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Create a Pull Request

### Development Guidelines

- Maintain comprehensive test coverage
- Follow Go conventions and formatting (`gofmt`, `golint`)
- Add benchmarks for performance-critical code
- Update documentation for new features
- Write clear commit messages

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Roadmap

- [ ] Enhanced circuit simulation with gate interconnection
- [ ] XNOR operation implementation  
- [ ] Contingency detection for logical functions
- [ ] Performance optimizations for large-scale operations
- [ ] Additional logical systems (fuzzy logic, modal logic)
- [ ] WebAssembly bindings
- [ ] CLI tool for interactive logic operations

## Changelog

### v1.0.0
- Initial release
- Basic boolean operations (AND, OR, XOR, NOT, NAND, NOR)
- Bitwise operations with individual bit manipulation
- Boolean vector support with element-wise operations
- Truth table generation with tautology/contradiction detection
- Basic circuit simulation and logic gate implementations
- Comprehensive test suite with benchmarks
- Fluent interface for operation chaining

---

**Made with ❤️ for the Go community**

If you find this package useful, please consider giving it a ⭐ on GitHub!