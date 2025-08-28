// Package classical provides high-performance boolean logic operations,
// bitwise manipulation, circuit simulation, truth table generation,
// and logical expression parsing/evaluation.
//
// This package offers a comprehensive set of tools for working with boolean
// logic in Go, including basic logical operations, vector operations,
// bitwise manipulation, logic gate simulation, truth table analysis,
// and AST-based expression evaluation.
//
// Basic usage:
//
//	result := logic.And(true, false, true) // false
//	vector := logic.NewBoolVector(true, false, true)
//	bits := logic.NewBitwiseInt(42)
//
//	// Expression evaluation
//	result, err := logic.EvaluateExpression("A & B | !C", map[string]bool{
//	    "A": true, "B": false, "C": true,
//	})
package classical

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/xDarkicex/logic/core"
)

// NodeType represents different types of AST nodes for logical expressions
type NodeType int

const (
	NodeVariable NodeType = iota
	NodeConstant
	NodeNot
	NodeAnd
	NodeOr
	NodeXor
	NodeNand
	NodeNor
	NodeImplies
	NodeIff
)

// String returns string representation of NodeType for debugging
func (nt NodeType) String() string {
	switch nt {
	case NodeVariable:
		return "Variable"
	case NodeConstant:
		return "Constant"
	case NodeNot:
		return "Not"
	case NodeAnd:
		return "And"
	case NodeOr:
		return "Or"
	case NodeXor:
		return "Xor"
	case NodeNand:
		return "Nand"
	case NodeNor:
		return "Nor"
	case NodeImplies:
		return "Implies"
	case NodeIff:
		return "Iff"
	default:
		return "Unknown"
	}
}

// ASTNode represents a node in the Abstract Syntax Tree for logical expressions.
// It forms the core structure for parsing and evaluating logical expressions.
type ASTNode struct {
	Type     NodeType
	Value    string     // Used for variables and constants
	Children []*ASTNode // Child nodes for operations
	Position int        // Position in original expression for error reporting
}

// EvaluationContext maps variable names to their boolean values
// for expression evaluation.
type EvaluationContext map[string]bool

// Evaluate evaluates the AST node with the given variable context.
// Returns the boolean result and any evaluation error.
func (node *ASTNode) Evaluate(ctx EvaluationContext) (bool, error) {
	switch node.Type {
	case NodeVariable:
		if value, exists := ctx[node.Value]; exists {
			return value, nil
		}
		return false, core.NewLogicError("classical", "ASTNode.Evaluate",
			fmt.Sprintf("undefined variable: %s", node.Value))

	case NodeConstant:
		lower := strings.ToLower(node.Value)
		return lower == "true" || lower == "1" || lower == "t", nil

	case NodeNot:
		if len(node.Children) != 1 {
			return false, core.NewLogicError("classical", "ASTNode.Evaluate", "NOT operation requires exactly one operand")
		}
		childVal, err := node.Children[0].Evaluate(ctx)
		if err != nil {
			return false, err
		}
		return Not(childVal), nil

	case NodeAnd:
		values := make([]bool, len(node.Children))
		for i, child := range node.Children {
			val, err := child.Evaluate(ctx)
			if err != nil {
				return false, err
			}
			values[i] = val
		}
		return And(values...), nil

	case NodeOr:
		values := make([]bool, len(node.Children))
		for i, child := range node.Children {
			val, err := child.Evaluate(ctx)
			if err != nil {
				return false, err
			}
			values[i] = val
		}
		return Or(values...), nil

	case NodeXor:
		values := make([]bool, len(node.Children))
		for i, child := range node.Children {
			val, err := child.Evaluate(ctx)
			if err != nil {
				return false, err
			}
			values[i] = val
		}
		return Xor(values...), nil

	case NodeNand:
		values := make([]bool, len(node.Children))
		for i, child := range node.Children {
			val, err := child.Evaluate(ctx)
			if err != nil {
				return false, err
			}
			values[i] = val
		}
		return Nand(values...), nil

	case NodeNor:
		values := make([]bool, len(node.Children))
		for i, child := range node.Children {
			val, err := child.Evaluate(ctx)
			if err != nil {
				return false, err
			}
			values[i] = val
		}
		return Nor(values...), nil

	case NodeImplies:
		if len(node.Children) != 2 {
			return false, core.NewLogicError("classical", "ASTNode.Evaluate", "IMPLIES operation requires exactly two operands")
		}
		left, err := node.Children[0].Evaluate(ctx)
		if err != nil {
			return false, err
		}
		right, err := node.Children[1].Evaluate(ctx)
		if err != nil {
			return false, err
		}
		return Implies(left, right), nil

	case NodeIff:
		if len(node.Children) != 2 {
			return false, core.NewLogicError("classical", "ASTNode.Evaluate", "IFF operation requires exactly two operands")
		}
		left, err := node.Children[0].Evaluate(ctx)
		if err != nil {
			return false, err
		}
		right, err := node.Children[1].Evaluate(ctx)
		if err != nil {
			return false, err
		}
		return Iff(left, right), nil

	default:
		return false, core.NewLogicError("classical", "ASTNode.Evaluate",
			fmt.Sprintf("unknown node type: %v", node.Type))
	}
}

// And performs logical AND operation on multiple boolean values.
// It returns true only if all inputs are true. If no inputs are provided,
// it returns false.
//
// Example:
//
//	And(true, true)        // true
//	And(true, false)       // false
//	And(true, true, true)  // true
//	And()                  // false
func And(inputs ...bool) bool {
	for _, v := range inputs {
		if !v {
			return false
		}
	}
	return len(inputs) > 0
}

// Or performs logical OR operation on multiple boolean values.
// It returns true if at least one input is true. If no inputs are provided,
// it returns false.
//
// Example:
//
//	Or(false, true)         // true
//	Or(false, false)        // false
//	Or(true, false, true)   // true
//	Or()                    // false
func Or(inputs ...bool) bool {
	for _, v := range inputs {
		if v {
			return true
		}
	}
	return false
}

// Xor performs exclusive OR operation on multiple boolean values.
// It returns true if an odd number of inputs are true.
//
// Example:
//
//	Xor(true, false)        // true
//	Xor(true, true)         // false
//	Xor(true, false, true)  // true
//	Xor(true, true, true)   // true
func Xor(inputs ...bool) bool {
	result := false
	for _, v := range inputs {
		if v {
			result = !result
		}
	}
	return result
}

// Not performs logical NOT operation on a single boolean value.
//
// Example:
//
//	Not(true)   // false
//	Not(false)  // true
func Not(input bool) bool {
	return !input
}

// Nand performs logical NAND (NOT AND) operation on multiple boolean values.
// It returns the negation of the AND operation.
//
// Example:
//
//	Nand(true, true)   // false
//	Nand(true, false)  // true
func Nand(inputs ...bool) bool {
	return !And(inputs...)
}

// Nor performs logical NOR (NOT OR) operation on multiple boolean values.
// It returns the negation of the OR operation.
//
// Example:
//
//	Nor(false, false)  // true
//	Nor(true, false)   // false
func Nor(inputs ...bool) bool {
	return !Or(inputs...)
}

// Xnor performs logical XNOR (exclusive NOR) operation on multiple boolean values.
// It returns true if an even number of inputs are true (opposite of XOR).
//
// Example:
//
//	Xnor(true, true)         // true
//	Xnor(true, false)        // false
//	Xnor(false, false)       // true
//	Xnor(true, true, true)   // false (odd number of trues)
func Xnor(inputs ...bool) bool {
	return !Xor(inputs...)
}

// Implies performs logical implication (A → B).
// It is equivalent to (!A || B). The implication is false only when
// the antecedent is true and the consequent is false.
//
// Example:
//
//	Implies(true, true)   // true
//	Implies(true, false)  // false
//	Implies(false, true)  // true
//	Implies(false, false) // true
func Implies(a, b bool) bool {
	return !a || b
}

// Iff performs logical biconditional (A ↔ B).
// It returns true if both inputs have the same truth value.
// This is also known as "if and only if".
//
// Example:
//
//	Iff(true, true)   // true
//	Iff(false, false) // true
//	Iff(true, false)  // false
func Iff(a, b bool) bool {
	return a == b
}

// DeMorganLaw verifies De Morgan's law: !(A && B) == (!A || !B).
// This function is used for testing logical equivalences and always returns true
// for valid inputs, demonstrating the law holds.
func DeMorganLaw(a, b bool) bool {
	return Not(And(a, b)) == Or(Not(a), Not(b))
}

// DistributiveLaw verifies the distributive law: A && (B || C) == (A && B) || (A && C).
// This function demonstrates that AND distributes over OR and always returns true
// for valid inputs.
func DistributiveLaw(a, b, c bool) bool {
	return And(a, Or(b, c)) == Or(And(a, b), And(a, c))
}

// BoolVector represents a vector of boolean values with optimized operations.
// It supports element-wise logical operations and various utility functions
// for working with collections of boolean values.
type BoolVector []bool

// NewBoolVector creates a new boolean vector from the given values.
// The vector is pre-allocated with the exact capacity to avoid reallocations.
//
// Example:
//
//	v := NewBoolVector(true, false, true)
//	fmt.Println(v) // [T, F, T]
func NewBoolVector(values ...bool) BoolVector {
	// Pre-allocate with exact capacity to avoid reallocations
	vec := make(BoolVector, len(values))
	copy(vec, values)
	return vec
}

// And performs element-wise AND operation with another vector.
// Both vectors must have the same length, otherwise an error is returned.
// The operation is optimized with loop unrolling for better performance.
//
// Example:
//
//	v1 := NewBoolVector(true, false, true)
//	v2 := NewBoolVector(false, true, true)
//	result, err := v1.And(v2) // [false, false, true], nil
func (bv BoolVector) And(other BoolVector) (BoolVector, error) {
	if len(bv) != len(other) {
		return nil, core.NewLogicError("classical", "BoolVector.And", "vector length mismatch")
	}

	result := make(BoolVector, len(bv))
	// Unroll loop for better performance on small vectors
	i := 0
	for ; i < len(bv)-3; i += 4 {
		result[i] = bv[i] && other[i]
		result[i+1] = bv[i+1] && other[i+1]
		result[i+2] = bv[i+2] && other[i+2]
		result[i+3] = bv[i+3] && other[i+3]
	}
	for ; i < len(bv); i++ {
		result[i] = bv[i] && other[i]
	}
	return result, nil
}

// Or performs element-wise OR operation with another vector.
// Both vectors must have the same length, otherwise an error is returned.
//
// Example:
//
//	v1 := NewBoolVector(true, false, false)
//	v2 := NewBoolVector(false, true, false)
//	result, err := v1.Or(v2) // [true, true, false], nil
func (bv BoolVector) Or(other BoolVector) (BoolVector, error) {
	if len(bv) != len(other) {
		return nil, core.NewLogicError("classical", "BoolVector.Or", "vector length mismatch")
	}

	result := make(BoolVector, len(bv))
	i := 0
	for ; i < len(bv)-3; i += 4 {
		result[i] = bv[i] || other[i]
		result[i+1] = bv[i+1] || other[i+1]
		result[i+2] = bv[i+2] || other[i+2]
		result[i+3] = bv[i+3] || other[i+3]
	}
	for ; i < len(bv); i++ {
		result[i] = bv[i] || other[i]
	}
	return result, nil
}

// Xor performs element-wise XOR operation with another vector.
// Both vectors must have the same length, otherwise an error is returned.
//
// Example:
//
//	v1 := NewBoolVector(true, false, true)
//	v2 := NewBoolVector(false, false, true)
//	result, err := v1.Xor(v2) // [true, false, false], nil
func (bv BoolVector) Xor(other BoolVector) (BoolVector, error) {
	if len(bv) != len(other) {
		return nil, core.NewLogicError("classical", "BoolVector.Xor", "vector length mismatch")
	}

	result := make(BoolVector, len(bv))
	i := 0
	for ; i < len(bv)-3; i += 4 {
		result[i] = bv[i] != other[i]
		result[i+1] = bv[i+1] != other[i+1]
		result[i+2] = bv[i+2] != other[i+2]
		result[i+3] = bv[i+3] != other[i+3]
	}
	for ; i < len(bv); i++ {
		result[i] = bv[i] != other[i]
	}
	return result, nil
}

// Not performs element-wise NOT operation on the vector.
// It returns a new vector with all boolean values inverted.
//
// Example:
//
//	v := NewBoolVector(true, false, true)
//	result := v.Not() // [false, true, false]
func (bv BoolVector) Not() BoolVector {
	result := make(BoolVector, len(bv))
	i := 0
	for ; i < len(bv)-3; i += 4 {
		result[i] = !bv[i]
		result[i+1] = !bv[i+1]
		result[i+2] = !bv[i+2]
		result[i+3] = !bv[i+3]
	}
	for ; i < len(bv); i++ {
		result[i] = !bv[i]
	}
	return result
}

// Count returns the number of true values in the vector.
// This is equivalent to the population count for boolean vectors.
//
// Example:
//
//	v := NewBoolVector(true, false, true, true)
//	count := v.Count() // 3
func (bv BoolVector) Count() int {
	count := 0
	for _, v := range bv {
		if v {
			count++
		}
	}
	return count
}

// AllTrue returns true if all values in the vector are true.
// An empty vector returns false.
//
// Example:
//
//	v1 := NewBoolVector(true, true, true)
//	v1.AllTrue() // true
//	v2 := NewBoolVector(true, false, true)
//	v2.AllTrue() // false
func (bv BoolVector) AllTrue() bool {
	for _, v := range bv {
		if !v {
			return false
		}
	}
	return len(bv) > 0
}

// AnyTrue returns true if any value in the vector is true.
// An empty vector returns false.
//
// Example:
//
//	v1 := NewBoolVector(false, true, false)
//	v1.AnyTrue() // true
//	v2 := NewBoolVector(false, false, false)
//	v2.AnyTrue() // false
func (bv BoolVector) AnyTrue() bool {
	for _, v := range bv {
		if v {
			return true
		}
	}
	return false
}

// String returns a string representation of the boolean vector.
// True values are represented as 'T' and false values as 'F'.
// The format is [T, F, T] for readability.
func (bv BoolVector) String() string {
	if len(bv) == 0 {
		return "[]"
	}

	// Pre-calculate size for better performance
	size := 1 + len(bv)*4 - 2 // [ + (T/F + ", ") * n - 2 + ]
	result := make([]byte, 0, size)
	result = append(result, '[')

	for i, v := range bv {
		if i > 0 {
			result = append(result, ',', ' ')
		}
		if v {
			result = append(result, 'T')
		} else {
			result = append(result, 'F')
		}
	}
	result = append(result, ']')

	return *(*string)(unsafe.Pointer(&result))
}

// Tautology checks if a logical expression is always true for all possible
// input combinations. It generates all possible truth value combinations
// for the given variables and tests the function with each combination.
//
// Example:
//
//	// Check if A OR NOT A is always true
//	variables := []string{"A"}
//	fn := func(inputs ...bool) bool {
//		return Or(inputs[0], Not(inputs[0]))
//	}
//	isTautology := Tautology(variables, fn) // true
func Tautology(variables []string, fn func(...bool) bool) bool {
	n := len(variables)
	for i := 0; i < (1 << n); i++ {
		inputs := make([]bool, n)
		for j := 0; j < n; j++ {
			inputs[j] = (i>>j)&1 == 1
		}
		if !fn(inputs...) {
			return false
		}
	}
	return true
}

// Contradiction checks if a logical expression is always false for all possible
// input combinations. It generates all possible truth value combinations
// for the given variables and tests the function with each combination.
//
// Example:
//
//	// Check if A AND NOT A is always false
//	variables := []string{"A"}
//	fn := func(inputs ...bool) bool {
//		return And(inputs[0], Not(inputs[0]))
//	}
//	isContradiction := Contradiction(variables, fn) // true
func Contradiction(variables []string, fn func(...bool) bool) bool {
	n := len(variables)
	for i := 0; i < (1 << n); i++ {
		inputs := make([]bool, n)
		for j := 0; j < n; j++ {
			inputs[j] = (i>>j)&1 == 1
		}
		if fn(inputs...) {
			return false
		}
	}
	return true
}

// Contingency checks if a logical expression is contingent (sometimes true, sometimes false)
// for the given input combinations. A contingent expression is neither a tautology nor a contradiction.
// This optimized version evaluates the function only once across all combinations.
//
// Example:
//
//	// Check if A AND B is contingent
//	variables := []string{"A", "B"}
//	fn := func(inputs ...bool) bool {
//		return And(inputs[0], inputs[1])
//	}
//	isContingent := Contingency(variables, fn) // true
func Contingency(variables []string, fn func(...bool) bool) bool {
	n := len(variables)
	hasTrue := false
	hasFalse := false

	for i := 0; i < (1 << n); i++ {
		inputs := make([]bool, n)
		for j := 0; j < n; j++ {
			inputs[j] = (i>>j)&1 == 1
		}

		result := fn(inputs...)
		if result {
			hasTrue = true
		} else {
			hasFalse = true
		}

		// Early termination: if we've seen both true and false, it's contingent
		if hasTrue && hasFalse {
			return true
		}
	}

	// If we only saw all true or all false, it's not contingent
	return false
}

// Public API functions for expression evaluation
// EvaluateExpression parses and evaluates a logical expression with given variables.
//
// Supported operators:
//   - AND: &, ∧, and, AND
//   - OR: |, ∨, or, OR
//   - NOT: !, ¬, not, NOT
//   - XOR: ^, ⊕, xor, XOR
//   - IMPLIES: ->, →, implies, IMPLIES
//   - IFF: <->, ↔, iff, IFF
//
// Example:
//
//	result, err := EvaluateExpression("(A & B) | !C", map[string]bool{
//	    "A": true, "B": false, "C": true,
//	})
func EvaluateExpression(expr string, variables map[string]bool) (bool, error) {
	ast, err := ParseExpression(expr)
	if err != nil {
		return false, err
	}
	return ast.Evaluate(EvaluationContext(variables))
}

// ValidateExpression checks if an expression is syntactically valid.
// Returns nil if valid, error with details if invalid.
func ValidateExpression(expr string) error {
	_, err := ParseExpression(expr)
	return err
}

// GenerateTruthTableFromExpression creates a truth table from a logical expression string.
// This integrates the expression parser with the existing truth table functionality.
//
// Example:
//
//	table, err := GenerateTruthTableFromExpression("A -> (B & C)", []string{"A", "B", "C"})
func GenerateTruthTableFromExpression(expr string, variables []string) (*TruthTable, error) {
	ast, err := ParseExpression(expr)
	if err != nil {
		return nil, err
	}

	return GenerateTruthTable(variables, func(inputs ...bool) bool {
		ctx := make(EvaluationContext)
		for i, variable := range variables {
			if i < len(inputs) {
				ctx[variable] = inputs[i]
			}
		}
		result, _ := ast.Evaluate(ctx)
		return result
	}), nil
}
