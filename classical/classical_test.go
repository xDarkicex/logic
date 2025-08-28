package classical

import (
	"fmt"
	"testing"
)

func TestBasicParsing(t *testing.T) {
	tests := []struct {
		expr     string
		vars     map[string]bool
		expected bool
	}{
		{"A", map[string]bool{"A": true}, true},
		{"A", map[string]bool{"A": false}, false},
		{"!A", map[string]bool{"A": true}, false},
		{"A & B", map[string]bool{"A": true, "B": true}, true},
		{"A & B", map[string]bool{"A": true, "B": false}, false},
		{"A | B", map[string]bool{"A": false, "B": true}, true},
		{"(A & B) | C", map[string]bool{"A": false, "B": true, "C": true}, true},
		{"A -> B", map[string]bool{"A": true, "B": false}, false},
		{"A <-> B", map[string]bool{"A": true, "B": true}, true},
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			result, err := EvaluateExpression(test.expr, test.vars)
			if err != nil {
				t.Fatalf("Error evaluating %s: %v", test.expr, err)
			}
			if result != test.expected {
				t.Errorf("Expression %s: expected %v, got %v", test.expr, test.expected, result)
			}
		})
	}
}

func TestUnicodeOperators(t *testing.T) {
	tests := []struct {
		expr     string
		vars     map[string]bool
		expected bool
	}{
		{"A ∧ B", map[string]bool{"A": true, "B": true}, true},
		{"A ∧ B", map[string]bool{"A": true, "B": false}, false},
		{"A ∨ B", map[string]bool{"A": false, "B": true}, true},
		{"A ∨ B", map[string]bool{"A": false, "B": false}, false},
		{"A ⊕ B", map[string]bool{"A": true, "B": false}, true},
		{"A ⊕ B", map[string]bool{"A": true, "B": true}, false},
		{"A → B", map[string]bool{"A": true, "B": false}, false},
		{"A → B", map[string]bool{"A": false, "B": true}, true},
		{"A ↔ B", map[string]bool{"A": true, "B": true}, true},
		{"A ↔ B", map[string]bool{"A": true, "B": false}, false},
		{"¬A", map[string]bool{"A": true}, false},
		{"¬A", map[string]bool{"A": false}, true},
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			result, err := EvaluateExpression(test.expr, test.vars)
			if err != nil {
				t.Fatalf("Error evaluating %s: %v", test.expr, err)
			}
			if result != test.expected {
				t.Errorf("Expression %s: expected %v, got %v", test.expr, test.expected, result)
			}
		})
	}
}

func TestComplexExpressions(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		vars     map[string]bool
		expected bool
	}{
		{
			"Complex AND/OR with ASCII",
			"(A & B) | (C & D)",
			map[string]bool{"A": true, "B": false, "C": true, "D": true},
			true,
		},
		{
			"Complex AND/OR with Unicode",
			"(A ∧ B) ∨ (C ∧ D)",
			map[string]bool{"A": true, "B": false, "C": true, "D": true},
			true,
		},
		{
			"Nested NOT operations",
			"!!A",
			map[string]bool{"A": true},
			true,
		},
		{
			"Unicode NOT nested",
			"¬¬A",
			map[string]bool{"A": true},
			true,
		},
		{
			"Mixed operators",
			"A ∧ (B | C) → D",
			map[string]bool{"A": true, "B": false, "C": true, "D": false},
			false,
		},
		{
			"De Morgan's law verification",
			"!(A & B) <-> (!A | !B)",
			map[string]bool{"A": true, "B": false},
			true,
		},
		{
			"Unicode De Morgan's law",
			"¬(A ∧ B) ↔ (¬A ∨ ¬B)",
			map[string]bool{"A": true, "B": false},
			true,
		},
		{
			"XOR with implications",
			"(A ^ B) -> (A | B)",
			map[string]bool{"A": true, "B": false},
			true,
		},
		{
			"Unicode XOR with implications",
			"(A ⊕ B) → (A ∨ B)",
			map[string]bool{"A": true, "B": false},
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := EvaluateExpression(test.expr, test.vars)
			if err != nil {
				t.Fatalf("Error evaluating %s: %v", test.expr, err)
			}
			if result != test.expected {
				t.Errorf("Expression %s: expected %v, got %v", test.expr, test.expected, result)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{"true", true},
		{"false", false},
		{"TRUE", true},
		{"FALSE", false},
		{"True", true},
		{"False", false},
		{"1", true},
		{"0", false},
		{"T", true},
		{"F", false},
		{"t", true},
		{"f", false},
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			result, err := EvaluateExpression(test.expr, map[string]bool{})
			if err != nil {
				t.Fatalf("Error evaluating %s: %v", test.expr, err)
			}
			if result != test.expected {
				t.Errorf("Expression %s: expected %v, got %v", test.expr, test.expected, result)
			}
		})
	}
}

func TestKeywordOperators(t *testing.T) {
	tests := []struct {
		expr     string
		vars     map[string]bool
		expected bool
	}{
		{"A and B", map[string]bool{"A": true, "B": true}, true},
		{"A or B", map[string]bool{"A": false, "B": true}, true},
		{"not A", map[string]bool{"A": true}, false},
		{"A xor B", map[string]bool{"A": true, "B": false}, true},
		{"A nand B", map[string]bool{"A": true, "B": true}, false},
		{"A nor B", map[string]bool{"A": false, "B": false}, true},
		{"A implies B", map[string]bool{"A": true, "B": false}, false},
		{"A iff B", map[string]bool{"A": true, "B": true}, true},
		{"A AND B", map[string]bool{"A": true, "B": true}, true},
		{"A OR B", map[string]bool{"A": false, "B": true}, true},
		{"NOT A", map[string]bool{"A": true}, false},
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			result, err := EvaluateExpression(test.expr, test.vars)
			if err != nil {
				t.Fatalf("Error evaluating %s: %v", test.expr, err)
			}
			if result != test.expected {
				t.Errorf("Expression %s: expected %v, got %v", test.expr, test.expected, result)
			}
		})
	}
}

func TestTruthTableFromExpression(t *testing.T) {
	table, err := GenerateTruthTableFromExpression("A -> B", []string{"A", "B"})
	if err != nil {
		t.Fatalf("Error generating truth table: %v", err)
	}

	expected := []bool{true, true, false, true}
	for i, row := range table.Rows {
		if row.Output != expected[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], row.Output)
		}
	}
}

func TestUnicodeTruthTableFromExpression(t *testing.T) {
	table, err := GenerateTruthTableFromExpression("A → B", []string{"A", "B"})
	if err != nil {
		t.Fatalf("Error generating truth table: %v", err)
	}

	expected := []bool{true, true, false, true}
	for i, row := range table.Rows {
		if row.Output != expected[i] {
			t.Errorf("Row %d: expected %v, got %v", i, expected[i], row.Output)
		}
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"Missing operand", "A &"},
		{"Unmatched parentheses", "(A & B"},
		{"Invalid character", "A # B"},
		{"Empty expression", ""},
		{"Just operator", "&"},
		{"Multiple operators", "A & & B"},
		{"Invalid variable", "A & 123invalid"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := EvaluateExpression(test.expr, map[string]bool{"A": true, "B": false})
			if err == nil {
				t.Errorf("Expected error for expression: %s", test.expr)
			}
		})
	}
}

func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		expr     string
		vars     map[string]bool
		expected bool
	}{
		// NOT has highest precedence
		{"!A & B", map[string]bool{"A": false, "B": true}, true},
		{"¬A ∧ B", map[string]bool{"A": false, "B": true}, true},

		// AND has higher precedence than OR
		{"A | B & C", map[string]bool{"A": false, "B": true, "C": false}, false},
		{"A ∨ B ∧ C", map[string]bool{"A": false, "B": true, "C": false}, false},

		// Parentheses override precedence
		{"(A | B) & C", map[string]bool{"A": false, "B": true, "C": false}, false},
		{"(A ∨ B) ∧ C", map[string]bool{"A": false, "B": true, "C": false}, false},

		// Implication has lower precedence
		{"A & B -> C", map[string]bool{"A": true, "B": true, "C": false}, false},
		{"A ∧ B → C", map[string]bool{"A": true, "B": true, "C": false}, false},
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			result, err := EvaluateExpression(test.expr, test.vars)
			if err != nil {
				t.Fatalf("Error evaluating %s: %v", test.expr, err)
			}
			if result != test.expected {
				t.Errorf("Expression %s: expected %v, got %v", test.expr, test.expected, result)
			}
		})
	}
}

func TestValidateExpression(t *testing.T) {
	validTests := []string{
		"A",
		"A & B",
		"A ∧ B",
		"!A",
		"¬A",
		"(A | B) & C",
		"A -> B",
		"A → B",
		"A <-> B",
		"A ↔ B",
		"true",
		"false",
	}

	for _, expr := range validTests {
		t.Run("Valid: "+expr, func(t *testing.T) {
			err := ValidateExpression(expr)
			if err != nil {
				t.Errorf("Expression %s should be valid, got error: %v", expr, err)
			}
		})
	}

	invalidTests := []string{
		"A &",
		"& B",
		"(A",
		"A)",
		"A # B",
		"",
		"A && B", // Double operators not supported
		"A || B", // Double operators not supported
	}

	for _, expr := range invalidTests {
		t.Run("Invalid: "+expr, func(t *testing.T) {
			err := ValidateExpression(expr)
			if err == nil {
				t.Errorf("Expression %s should be invalid", expr)
			}
		})
	}
}

func TestVariableNames(t *testing.T) {
	tests := []struct {
		expr     string
		vars     map[string]bool
		expected bool
	}{
		{"Variable1", map[string]bool{"Variable1": true}, true},
		{"var_2", map[string]bool{"var_2": false}, false},
		{"X1 & Y2", map[string]bool{"X1": true, "Y2": true}, true},
		{"input_a | input_b", map[string]bool{"input_a": false, "input_b": true}, true},
		{"P ∧ Q", map[string]bool{"P": true, "Q": false}, false},
	}

	for _, test := range tests {
		t.Run(test.expr, func(t *testing.T) {
			result, err := EvaluateExpression(test.expr, test.vars)
			if err != nil {
				t.Fatalf("Error evaluating %s: %v", test.expr, err)
			}
			if result != test.expected {
				t.Errorf("Expression %s: expected %v, got %v", test.expr, test.expected, result)
			}
		})
	}
}

func TestUndefinedVariables(t *testing.T) {
	tests := []string{
		"A",
		"A & B",
		"X | Y | Z",
	}

	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			_, err := EvaluateExpression(expr, map[string]bool{})
			if err == nil {
				t.Errorf("Expected error for undefined variables in: %s", expr)
			}

			// Check that it's specifically a LogicError about undefined variable
			if logicErr, ok := err.(*LogicError); ok {
				if logicErr.Op != "ASTNode.Evaluate" {
					t.Errorf("Expected ASTNode.Evaluate error, got: %s", logicErr.Op)
				}
			} else {
				t.Errorf("Expected LogicError, got: %T", err)
			}
		})
	}
}

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
		{XnorGate{}, []bool{true, false}, false},
		{XnorGate{}, []bool{true, true}, true},
		{XnorGate{}, []bool{false, false}, true},
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
// ENHANCED CIRCUIT TESTS
// =======================
func TestEnhancedCircuit(t *testing.T) {
	// Create a simple AND-OR circuit: (A AND B) OR (C AND D)
	circuit := NewCircuit([]string{"A", "B", "C", "D"})

	// Add nodes
	err := circuit.AddNode("and1", AndGate{}, []string{"A", "B"})
	if err != nil {
		t.Fatalf("Failed to add and1: %v", err)
	}

	err = circuit.AddNode("and2", AndGate{}, []string{"C", "D"})
	if err != nil {
		t.Fatalf("Failed to add and2: %v", err)
	}

	err = circuit.AddNode("or1", OrGate{}, []string{"and1", "and2"})
	if err != nil {
		t.Fatalf("Failed to add or1: %v", err)
	}

	// Set output
	err = circuit.SetOutputs([]string{"or1"})
	if err != nil {
		t.Fatalf("Failed to set outputs: %v", err)
	}

	// Test cases
	testCases := []struct {
		name     string
		inputs   map[string]bool
		expected bool
	}{
		{"All false", map[string]bool{"A": false, "B": false, "C": false, "D": false}, false},
		{"A,B true", map[string]bool{"A": true, "B": true, "C": false, "D": false}, true},
		{"C,D true", map[string]bool{"A": false, "B": false, "C": true, "D": true}, true},
		{"All true", map[string]bool{"A": true, "B": true, "C": true, "D": true}, true},
		{"Mixed", map[string]bool{"A": true, "B": false, "C": false, "D": true}, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			outputs, err := circuit.Simulate(tc.inputs)
			if err != nil {
				t.Fatalf("Simulation failed: %v", err)
			}

			result := outputs["or1"]
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestCircuitMultipleOutputs(t *testing.T) {
	// Create circuit with multiple outputs
	circuit := NewCircuit([]string{"A", "B"})

	circuit.AddNode("and1", AndGate{}, []string{"A", "B"})
	circuit.AddNode("or1", OrGate{}, []string{"A", "B"})
	circuit.AddNode("xor1", XorGate{}, []string{"A", "B"})
	circuit.AddNode("not1", NotGate{}, []string{"A"})

	circuit.SetOutputs([]string{"and1", "or1", "xor1", "not1"})

	inputs := map[string]bool{"A": true, "B": false}
	outputs, err := circuit.Simulate(inputs)
	if err != nil {
		t.Fatalf("Simulation failed: %v", err)
	}

	expected := map[string]bool{
		"and1": false, // true AND false
		"or1":  true,  // true OR false
		"xor1": true,  // true XOR false
		"not1": false, // NOT true
	}

	for nodeID, expectedValue := range expected {
		if outputs[nodeID] != expectedValue {
			t.Errorf("Node %s: expected %v, got %v", nodeID, expectedValue, outputs[nodeID])
		}
	}
}

func TestCircuitErrorHandling(t *testing.T) {
	circuit := NewCircuit([]string{"A", "B"})

	// Test duplicate node ID
	circuit.AddNode("gate1", AndGate{}, []string{"A", "B"})
	err := circuit.AddNode("gate1", OrGate{}, []string{"A", "B"})
	if err == nil {
		t.Error("Expected error for duplicate node ID")
	}

	// Test invalid output reference
	err = circuit.SetOutputs([]string{"nonexistent"})
	if err == nil {
		t.Error("Expected error for nonexistent output node")
	}

	// Test missing input
	circuit.SetOutputs([]string{"gate1"})
	_, err = circuit.Simulate(map[string]bool{"A": true}) // Missing B
	if err == nil {
		t.Error("Expected error for missing input")
	}
}

func TestCircuitCyclicDependency(t *testing.T) {
	circuit := NewCircuit([]string{"A"})

	// Create a cycle: gate1 -> gate2 -> gate1
	circuit.AddNode("gate1", AndGate{}, []string{"A", "gate2"})
	circuit.AddNode("gate2", OrGate{}, []string{"gate1", "A"})
	circuit.SetOutputs([]string{"gate1"})

	// This should detect the cycle
	_, err := circuit.Simulate(map[string]bool{"A": true})
	if err == nil {
		t.Error("Expected error for cyclic dependency")
	}
}

func TestCircuitComplexTopology(t *testing.T) {
	// Test a more complex circuit with deeper dependencies
	circuit := NewCircuit([]string{"A", "B", "C", "D"})

	// Layer 1
	circuit.AddNode("gate1", AndGate{}, []string{"A", "B"})
	circuit.AddNode("gate2", OrGate{}, []string{"C", "D"})

	// Layer 2
	circuit.AddNode("gate3", XorGate{}, []string{"gate1", "gate2"})
	circuit.AddNode("gate4", NandGate{}, []string{"A", "C"})

	// Layer 3
	circuit.AddNode("final", OrGate{}, []string{"gate3", "gate4"})

	circuit.SetOutputs([]string{"final"})

	inputs := map[string]bool{"A": true, "B": false, "C": true, "D": false}
	outputs, err := circuit.Simulate(inputs)
	if err != nil {
		t.Fatalf("Complex circuit simulation failed: %v", err)
	}

	// Verify intermediate values
	gate1Val, _ := circuit.GetNodeValue("gate1") // true AND false = false
	gate2Val, _ := circuit.GetNodeValue("gate2") // true OR false = true
	gate3Val, _ := circuit.GetNodeValue("gate3") // false XOR true = true
	gate4Val, _ := circuit.GetNodeValue("gate4") // NOT(true AND true) = false

	if gate1Val != false {
		t.Errorf("gate1: expected false, got %v", gate1Val)
	}
	if gate2Val != true {
		t.Errorf("gate2: expected true, got %v", gate2Val)
	}
	if gate3Val != true {
		t.Errorf("gate3: expected true, got %v", gate3Val)
	}
	if gate4Val != false {
		t.Errorf("gate4: expected false, got %v", gate4Val)
	}

	// Final output: true OR false = true
	if outputs["final"] != true {
		t.Errorf("final: expected true, got %v", outputs["final"])
	}
}

func TestXnorGate(t *testing.T) {
	gate := XnorGate{}

	tests := []struct {
		inputs   []bool
		expected bool
	}{
		{[]bool{false, false}, true},
		{[]bool{false, true}, false},
		{[]bool{true, false}, false},
		{[]bool{true, true}, true},
		{[]bool{true, true, true}, false},        // Odd number of trues
		{[]bool{true, true, false, false}, true}, // Even number of trues
	}

	for _, test := range tests {
		result := gate.Evaluate(test.inputs...)
		if result != test.expected {
			t.Errorf("XNOR(%v): expected %v, got %v", test.inputs, test.expected, result)
		}
	}
}

func TestContingency(t *testing.T) {
	variables := []string{"A", "B"}

	// A AND B should be contingent
	andFn := func(inputs ...bool) bool {
		return And(inputs[0], inputs[1])
	}
	if !Contingency(variables, andFn) {
		t.Error("A AND B should be contingent")
	}

	// A OR NOT A should be a tautology (not contingent)
	tautologyFn := func(inputs ...bool) bool {
		return Or(inputs[0], Not(inputs[0]))
	}
	if Contingency([]string{"A"}, tautologyFn) {
		t.Error("A OR NOT A should not be contingent (it's a tautology)")
	}

	// A AND NOT A should be a contradiction (not contingent)
	contradictionFn := func(inputs ...bool) bool {
		return And(inputs[0], Not(inputs[0]))
	}
	if Contingency([]string{"A"}, contradictionFn) {
		t.Error("A AND NOT A should not be contingent (it's a contradiction)")
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

func BenchmarkCircuitSimulation(b *testing.B) {
	// Benchmark circuit simulation
	circuit := NewCircuit([]string{"A", "B", "C", "D"})

	circuit.AddNode("and1", AndGate{}, []string{"A", "B"})
	circuit.AddNode("and2", AndGate{}, []string{"C", "D"})
	circuit.AddNode("or1", OrGate{}, []string{"and1", "and2"})
	circuit.AddNode("not1", NotGate{}, []string{"or1"})

	circuit.SetOutputs([]string{"not1"})

	inputs := map[string]bool{"A": true, "B": false, "C": true, "D": true}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		circuit.Simulate(inputs)
	}
}
