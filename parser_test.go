package logic

import (
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
