package logic

import (
	"fmt"
	"testing"
)

// =======================
// BASIC OPERATION TESTS
// =======================
func TestBasicOperations(t *testing.T) {
	tests := []struct {
		name     string
		inputs   []bool
		expected bool
		op       func(...bool) bool
	}{
		{"AND true,true", []bool{true, true}, true, And},
		{"AND true,false", []bool{true, false}, false, And},
		{"OR true,false", []bool{true, false}, true, Or},
		{"OR false,false", []bool{false, false}, false, Or},
		{"XOR true,false", []bool{true, false}, true, Xor},
		{"XOR true,true", []bool{true, true}, false, Xor},
		{"NAND true,true", []bool{true, true}, false, Nand},
		{"NOR false,false", []bool{false, false}, true, Nor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.op(tt.inputs...)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestImplications(t *testing.T) {
	tests := []struct {
		a, b    bool
		implies bool
		iff     bool
	}{
		{true, true, true, true},
		{true, false, false, false},
		{false, true, true, false},
		{false, false, true, true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("A=%v,B=%v", tt.a, tt.b), func(t *testing.T) {
			if Implies(tt.a, tt.b) != tt.implies {
				t.Errorf("Implies(%v,%v): expected %v", tt.a, tt.b, tt.implies)
			}
			if Iff(tt.a, tt.b) != tt.iff {
				t.Errorf("Iff(%v,%v): expected %v", tt.a, tt.b, tt.iff)
			}
		})
	}
}

// =======================
// BOOLEAN VECTOR TESTS
// =======================
func TestBoolVector(t *testing.T) {
	v1 := NewBoolVector(true, false, true)
	v2 := NewBoolVector(false, false, true)

	// Test AND
	result, err := v1.And(v2)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	expected := NewBoolVector(false, false, true)
	for i, v := range result {
		if v != expected[i] {
			t.Errorf("AND operation failed at index %d", i)
		}
	}

	// Test count operations
	if v1.Count() != 2 {
		t.Errorf("Expected count 2, got %d", v1.Count())
	}
	if !v1.AnyTrue() {
		t.Error("Expected AnyTrue to be true")
	}
	if v1.AllTrue() {
		t.Error("Expected AllTrue to be false")
	}
}

// =======================
// BITWISE OPERATION TESTS
// =======================
func TestBitwiseOperations(t *testing.T) {
	a := NewBitwiseInt(0b1010) // 10
	b := NewBitwiseInt(0b1100) // 12

	tests := []struct {
		name     string
		result   BitwiseInt
		expected uint64
	}{
		{"AND", a.And(b), 0b1000}, // 8
		{"OR", a.Or(b), 0b1110},   // 14
		{"XOR", a.Xor(b), 0b0110}, // 6
		{"NOT a", a.Not(), ^uint64(0b1010)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.Value() != tt.expected {
				t.Errorf("Expected %b, got %b", tt.expected, tt.result.Value())
			}
		})
	}
}

func TestBitManipulation(t *testing.T) {
	bi := NewBitwiseInt(0b1010) // 10

	// Test bit operations
	setBit := bi.SetBit(0)       // Should be 0b1011 (11)
	clearBit := bi.ClearBit(1)   // Should be 0b1000 (8)
	toggleBit := bi.ToggleBit(2) // Should be 0b1110 (14)

	if setBit.Value() != 11 {
		t.Errorf("SetBit failed: expected 11, got %d", setBit.Value())
	}
	if clearBit.Value() != 8 {
		t.Errorf("ClearBit failed: expected 8, got %d", clearBit.Value())
	}
	if toggleBit.Value() != 14 {
		t.Errorf("ToggleBit failed: expected 14, got %d", toggleBit.Value())
	}

	// Test bit queries
	if !bi.GetBit(1) {
		t.Error("GetBit(1) should be true")
	}
	if bi.GetBit(0) {
		t.Error("GetBit(0) should be false")
	}

	// Test count set bits
	if bi.CountSetBits() != 2 {
		t.Errorf("Expected 2 set bits, got %d", bi.CountSetBits())
	}

	// Test power of 2
	powerOf2 := NewBitwiseInt(8)
	if !powerOf2.IsPowerOfTwo() {
		t.Error("8 should be a power of 2")
	}
	if bi.IsPowerOfTwo() {
		t.Error("10 should not be a power of 2")
	}
}

// =======================
// TRUTH TABLE TESTS
// =======================
func TestTruthTable(t *testing.T) {
	// Test AND truth table
	andTable := GenerateTruthTable(
		[]string{"A", "B"},
		func(inputs ...bool) bool {
			return And(inputs...)
		},
	)

	expectedOutputs := []bool{false, false, false, true}
	for i, row := range andTable.Rows {
		if row.Output != expectedOutputs[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expectedOutputs[i], row.Output)
		}
	}
}

// =======================
// LOGICAL LAWS TESTS
// =======================
func TestLogicalLaws(t *testing.T) {
	// Test De Morgan's Law
	variables := []string{"A", "B"}
	deMorganLaw := func(inputs ...bool) bool {
		return DeMorganLaw(inputs[0], inputs[1])
	}

	if !Tautology(variables, deMorganLaw) {
		t.Error("De Morgan's law should be a tautology")
	}

	// Test Distributive Law
	distributiveLaw := func(inputs ...bool) bool {
		if len(inputs) < 3 {
			return false
		}
		return DistributiveLaw(inputs[0], inputs[1], inputs[2])
	}

	variables3 := []string{"A", "B", "C"}
	if !Tautology(variables3, distributiveLaw) {
		t.Error("Distributive law should be a tautology")
	}
}

// =======================
// EVALUATOR TESTS
// =======================
func TestEvaluator(t *testing.T) {
	// Test fluent interface
	result := Eval(true).And(false).Or(true).Xor(false).Result()
	if !result {
		t.Error("Evaluator chain failed")
	}

	// Test NOT operation
	notResult := Eval(true).Not().Result()
	if notResult {
		t.Error("NOT operation failed")
	}
}

// =======================
// GATE TESTS
// =======================
func TestGates(t *testing.T) {
	gates := []struct {
		gate     Gate
		inputs   []bool
		expected bool
	}{
		{AndGate{}, []bool{true, true}, true},
		{AndGate{}, []bool{true, false}, false},
		{OrGate{}, []bool{false, true}, true},
		{OrGate{}, []bool{false, false}, false},
		{NotGate{}, []bool{true}, false},
		{NotGate{}, []bool{false}, true},
		{XorGate{}, []bool{true, false}, true},
		{XorGate{}, []bool{true, true}, false},
		{NandGate{}, []bool{true, true}, false},
		{NandGate{}, []bool{true, false}, true},
	}

	for _, test := range gates {
		t.Run(test.gate.String(), func(t *testing.T) {
			result := test.gate.Evaluate(test.inputs...)
			if result != test.expected {
				t.Errorf("Gate %s with inputs %v: expected %v, got %v",
					test.gate.String(), test.inputs, test.expected, result)
			}
		})
	}
}

// =======================
// ERROR HANDLING TESTS
// =======================
func TestErrorHandling(t *testing.T) {
	v1 := NewBoolVector(true, false)
	v2 := NewBoolVector(true, false, true) // Different length

	// Test vector length mismatch
	_, err := v1.And(v2)
	if err == nil {
		t.Error("Expected error for mismatched vector lengths")
	}

	logicErr, ok := err.(*LogicError)
	if !ok {
		t.Error("Expected LogicError type")
	}
	if logicErr.Op != "BoolVector.And" {
		t.Errorf("Expected operation 'BoolVector.And', got '%s'", logicErr.Op)
	}
}

// =======================
// PERFORMANCE COMPARISON TESTS
// =======================
func TestPerformanceComparison(t *testing.T) {
	// Compare built-in Go operators vs our package for simple operations
	benchmark := NewBenchmark()

	// Add operations to benchmark
	benchmark.Add("Built-in AND", func() bool { return true && false })
	benchmark.Add("Package AND", func() bool { return And(true, false) })
	benchmark.Add("Built-in OR", func() bool { return true || false })
	benchmark.Add("Package OR", func() bool { return Or(true, false) })

	// Run benchmark
	benchmark.Run()

	// Verify results are correct
	expected := []bool{false, false, true, true}
	for i, result := range benchmark.Results {
		if result != expected[i] {
			t.Errorf("Benchmark operation %d: expected %v, got %v",
				i, expected[i], result)
		}
	}
}

// =======================
// BENCHMARK TESTS
// =======================
func BenchmarkBasicOperations(b *testing.B) {
	for i := 0; i < b.N; i++ {
		And(true, false, true)
		Or(false, true, false)
		Xor(true, false, true)
		Not(true)
	}
}

func BenchmarkBitwiseOperations(b *testing.B) {
	a := NewBitwiseInt(0xAAAAAAAA)
	c := NewBitwiseInt(0x55555555)
	for i := 0; i < b.N; i++ {
		a.And(c)
		a.Or(c)
		a.Xor(c)
		a.Not()
	}
}

func BenchmarkBoolVectorOperations(b *testing.B) {
	v1 := NewBoolVector(true, false, true, false, true, false, true, false)
	v2 := NewBoolVector(false, true, false, true, false, true, false, true)
	for i := 0; i < b.N; i++ {
		v1.And(v2)
		v1.Or(v2)
		v1.Xor(v2)
		v1.Not()
	}
}

func BenchmarkTruthTableGeneration(b *testing.B) {
	variables := []string{"A", "B", "C"}
	fn := func(inputs ...bool) bool {
		return And(Or(inputs[0], inputs[1]), inputs[2])
	}
	for i := 0; i < b.N; i++ {
		GenerateTruthTable(variables, fn)
	}
}
