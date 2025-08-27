// Package main demonstrates usage examples for the logic package.
// This file contains comprehensive examples showing how to use all
// the major features of the logic package.
package main

import (
	"fmt"

	"github.com/xDarkicex/logic"
)

// ExampleBasicOperations demonstrates basic boolean operations.
// Shows how to use And, Or, Xor, and Not functions with various inputs.
func ExampleBasicOperations() {
	fmt.Println("=== Basic Boolean Operations ===")

	// Basic boolean operations
	fmt.Printf("AND(true, false): %v\n", logic.And(true, false))
	fmt.Printf("OR(true, false): %v\n", logic.Or(true, false))
	fmt.Printf("XOR(true, false): %v\n", logic.Xor(true, false))
	fmt.Printf("NOT(true): %v\n", logic.Not(true))

	// Multiple inputs
	fmt.Printf("AND(true, true, false): %v\n", logic.And(true, true, false))
	fmt.Printf("OR(false, false, true): %v\n", logic.Or(false, false, true))

	// Advanced operations
	fmt.Printf("NAND(true, true): %v\n", logic.Nand(true, true))
	fmt.Printf("NOR(false, false): %v\n", logic.Nor(false, false))
	fmt.Printf("IMPLIES(true, false): %v\n", logic.Implies(true, false))
	fmt.Printf("IFF(true, true): %v\n", logic.Iff(true, true))

	fmt.Println()
}

// ExampleBoolVector demonstrates boolean vector operations.
// Shows how to create vectors and perform element-wise operations.
func ExampleBoolVector() {
	fmt.Println("=== Boolean Vector Operations ===")

	// Create boolean vectors
	v1 := logic.NewBoolVector(true, false, true, false)
	v2 := logic.NewBoolVector(false, true, true, false)

	fmt.Printf("Vector 1: %s\n", v1.String())
	fmt.Printf("Vector 2: %s\n", v2.String())

	// Perform operations
	and, _ := v1.And(v2)
	or, _ := v1.Or(v2)
	xor, _ := v1.Xor(v2)

	fmt.Printf("V1 AND V2: %s\n", and.String())
	fmt.Printf("V1 OR V2: %s\n", or.String())
	fmt.Printf("V1 XOR V2: %s\n", xor.String())
	fmt.Printf("NOT V1: %s\n", v1.Not().String())

	// Statistics
	fmt.Printf("V1 count: %d\n", v1.Count())
	fmt.Printf("V1 all true: %v\n", v1.AllTrue())
	fmt.Printf("V1 any true: %v\n", v1.AnyTrue())

	fmt.Println()
}

// ExampleBitwiseOperations demonstrates bitwise operations.
// Shows how to work with BitwiseInt for bit manipulation.
func ExampleBitwiseOperations() {
	fmt.Println("=== Bitwise Operations ===")

	// Create bitwise integers
	a := logic.NewBitwiseInt(0b1010) // 10
	b := logic.NewBitwiseInt(0b1100) // 12

	fmt.Printf("A: %s\n", a.String())
	fmt.Printf("B: %s\n", b.String())

	// Basic bitwise operations
	fmt.Printf("A AND B: %s\n", a.And(b).String())
	fmt.Printf("A OR B: %s\n", a.Or(b).String())
	fmt.Printf("A XOR B: %s\n", a.Xor(b).String())

	// Bit manipulation
	fmt.Printf("A set bit 0: %s\n", a.SetBit(0).String())
	fmt.Printf("A clear bit 1: %s\n", a.ClearBit(1).String())
	fmt.Printf("A toggle bit 2: %s\n", a.ToggleBit(2).String())

	// Bit queries
	fmt.Printf("A get bit 1: %v\n", a.GetBit(1))
	fmt.Printf("A count set bits: %d\n", a.CountSetBits())
	fmt.Printf("A is power of 2: %v\n", a.IsPowerOfTwo())

	// Shifting
	fmt.Printf("A left shift 2: %s\n", a.LeftShift(2).String())
	fmt.Printf("A right shift 1: %s\n", a.RightShift(1).String())

	fmt.Println()
}

// ExampleTruthTable demonstrates truth table generation.
// Shows how to create and display truth tables for logical functions.
func ExampleTruthTable() {
	fmt.Println("=== Truth Table Generation ===")

	// Generate truth table for XOR
	xorTable := logic.GenerateTruthTable(
		[]string{"A", "B"},
		func(inputs ...bool) bool {
			return logic.Xor(inputs...)
		},
	)

	fmt.Println("XOR Truth Table:")
	fmt.Print(xorTable.String())

	// Generate truth table for more complex operation: (A AND B) OR (NOT A AND NOT B)
	complexTable := logic.GenerateTruthTable(
		[]string{"A", "B"},
		func(inputs ...bool) bool {
			a, b := inputs[0], inputs[1]
			return logic.Or(logic.And(a, b), logic.And(logic.Not(a), logic.Not(b)))
		},
	)

	fmt.Println("Complex Operation Truth Table:")
	fmt.Print(complexTable.String())

	fmt.Println()
}

// ExampleLogicalLaws demonstrates logical laws verification.
// Shows how to verify fundamental logical laws using the tautology checker.
func ExampleLogicalLaws() {
	fmt.Println("=== Logical Laws Verification ===")

	// Demonstrate logical laws using tautology checker
	variables := []string{"A", "B"}

	// De Morgan's Law: !(A && B) = !A || !B
	deMorgan := func(inputs ...bool) bool {
		return logic.DeMorganLaw(inputs[0], inputs[1])
	}
	fmt.Printf("De Morgan's Law is a tautology: %v\n", logic.Tautology(variables, deMorgan))

	// Distributive Law: A && (B || C) = (A && B) || (A && C)
	variables3 := []string{"A", "B", "C"}
	distributive := func(inputs ...bool) bool {
		return logic.DistributiveLaw(inputs[0], inputs[1], inputs[2])
	}
	fmt.Printf("Distributive Law is a tautology: %v\n", logic.Tautology(variables3, distributive))

	// Law of excluded middle: A || !A (always true)
	excludedMiddle := func(inputs ...bool) bool {
		a := inputs[0]
		return logic.Or(a, logic.Not(a))
	}
	fmt.Printf("Law of excluded middle is a tautology: %v\n",
		logic.Tautology([]string{"A"}, excludedMiddle))

	// Contradiction: A && !A (always false)
	contradiction := func(inputs ...bool) bool {
		a := inputs[0]
		return logic.And(a, logic.Not(a))
	}
	fmt.Printf("A && !A is a contradiction: %v\n",
		logic.Contradiction([]string{"A"}, contradiction))

	fmt.Println()
}

// ExampleFluentInterface demonstrates the fluent evaluator interface.
// Shows how to chain logical operations using method chaining.
func ExampleFluentInterface() {
	fmt.Println("=== Fluent Interface ===")

	// Using the fluent evaluator interface
	result1 := logic.Eval(true).And(false).Or(true).Result()
	result2 := logic.Eval(false).Not().And(true).Xor(false).Result()

	fmt.Printf("Eval(true).And(false).Or(true): %v\n", result1)
	fmt.Printf("Eval(false).Not().And(true).Xor(false): %v\n", result2)

	// Chain complex operations
	complex := logic.Eval(true).
		And(false). // false
		Or(true).   // true
		Xor(false). // true
		And(true).  // true
		Not().      // false
		Or(true).   // true
		Result()

	fmt.Printf("Complex chain result: %v\n", complex)

	fmt.Println()
}

// ExampleGatesAndCircuits demonstrates logic gates and circuits.
// Shows how to use individual gates and simple circuit simulation.
func ExampleGatesAndCircuits() {
	fmt.Println("=== Logic Gates and Circuits ===")

	// Test individual gates
	andGate := logic.AndGate{}
	orGate := logic.OrGate{}
	notGate := logic.NotGate{}
	xorGate := logic.XorGate{}
	nandGate := logic.NandGate{}

	fmt.Printf("AND(true, false): %v\n", andGate.Evaluate(true, false))
	fmt.Printf("OR(true, false): %v\n", orGate.Evaluate(true, false))
	fmt.Printf("NOT(true): %v\n", notGate.Evaluate(true))
	fmt.Printf("XOR(true, false): %v\n", xorGate.Evaluate(true, false))
	fmt.Printf("NAND(true, true): %v\n", nandGate.Evaluate(true, true))

	// Create a simple circuit
	circuit := logic.NewCircuit([]string{"A", "B"})
	circuit.AddGate(andGate)
	circuit.AddGate(orGate)

	inputs := map[string]bool{"A": true, "B": false}
	result, err := circuit.Simulate(inputs)
	if err != nil {
		fmt.Printf("Circuit simulation error: %v\n", err)
	} else {
		fmt.Printf("Circuit result: %v\n", result)
	}

	fmt.Println()
}

// ExampleAdvancedBitwiseOperations demonstrates advanced bitwise operations.
// Shows complex bit manipulation patterns and conversions.
func ExampleAdvancedBitwiseOperations() {
	fmt.Println("=== Advanced Bitwise Operations ===")

	// Demonstrate advanced bitwise operations
	num := logic.NewBitwiseInt(42) // 101010 in binary
	fmt.Printf("Original number: %s\n", num.String())

	// Population count (number of 1 bits)
	fmt.Printf("Population count: %d\n", num.CountSetBits())

	// Check if power of 2
	fmt.Printf("Is power of 2: %v\n", num.IsPowerOfTwo())

	// Convert to boolean vector and back
	boolVec := num.ToBoolVector()
	fmt.Printf("As boolean vector (first 8 bits): %v\n", boolVec[:8])

	// Bit manipulation patterns
	// Set all even bits
	evenBits := logic.NewBitwiseInt(0)
	for i := uint(0); i < 64; i += 2 {
		evenBits = evenBits.SetBit(i)
	}
	fmt.Printf("Even bits pattern (first 16 bits): 0b%016b\n", evenBits.Value()&0xFFFF)

	// Isolate rightmost set bit
	rightmostBit := num.And(logic.NewBitwiseInt(^num.Value() + 1))
	fmt.Printf("Rightmost set bit: 0b%b\n", rightmostBit.Value())

	fmt.Println()
}

// ExampleErrorHandling demonstrates error handling in the package.
// Shows how errors are reported and handled.
func ExampleErrorHandling() {
	fmt.Println("=== Error Handling ===")

	// Create vectors of different lengths
	v1 := logic.NewBoolVector(true, false)
	v2 := logic.NewBoolVector(true, false, true) // Different length

	// This will generate an error
	_, err := v1.And(v2)
	if err != nil {
		fmt.Printf("Error occurred: %v\n", err)

		// Check if it's a LogicError
		if logicErr, ok := err.(*logic.LogicError); ok {
			fmt.Printf("Operation: %s\n", logicErr.Op)
			fmt.Printf("Message: %s\n", logicErr.Message)
		}
	}

	fmt.Println()
}

// main runs all the examples to demonstrate the logic package capabilities.
func main() {
	fmt.Println("Logic Package Examples")
	fmt.Println("======================")
	fmt.Println()

	ExampleBasicOperations()
	ExampleBoolVector()
	ExampleBitwiseOperations()
	ExampleTruthTable()
	ExampleLogicalLaws()
	ExampleFluentInterface()
	ExampleGatesAndCircuits()
	ExampleAdvancedBitwiseOperations()
	ExampleErrorHandling()

	fmt.Println("All examples completed successfully!")
}
