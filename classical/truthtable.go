package classical

import (
	"fmt"
	"strings"
)

// TruthTableRow represents a single row in a truth table.
// It contains the input variable assignments and the resulting output
// for a specific combination of input values.
type TruthTableRow struct {
	// Inputs maps variable names to their boolean values for this row
	Inputs map[string]bool

	// Output is the result of the logical function for this input combination
	Output bool
}

// TruthTable represents a complete truth table for a logical function.
// It contains all possible input combinations and their corresponding outputs
// for a given set of boolean variables.
type TruthTable struct {
	// Variables contains the names of the input variables in order
	Variables []string

	// Rows contains all possible input/output combinations
	Rows []TruthTableRow
}

// GenerateTruthTable creates a truth table for the given variables and function.
// It systematically generates all possible combinations of input values
// (2^n combinations for n variables) and evaluates the function for each.
//
// The function fn should accept a slice of boolean values corresponding
// to the variables in order and return the logical result.
//
// Example:
//
//	// Create truth table for AND operation
//	variables := []string{"A", "B"}
//	andFunction := func(inputs ...bool) bool {
//		return And(inputs...)
//	}
//	table := GenerateTruthTable(variables, andFunction)
func GenerateTruthTable(variables []string, fn func(...bool) bool) *TruthTable {
	n := len(variables)
	numRows := 1 << n

	table := &TruthTable{
		Variables: make([]string, len(variables)),
		Rows:      make([]TruthTableRow, numRows),
	}
	copy(table.Variables, variables)

	for i := 0; i < numRows; i++ {
		inputs := make(map[string]bool, n)
		inputSlice := make([]bool, n)

		// Generate binary representation for this row
		for j := 0; j < n; j++ {
			value := (i>>(n-1-j))&1 == 1
			inputs[variables[j]] = value
			inputSlice[j] = value
		}

		table.Rows[i] = TruthTableRow{
			Inputs: inputs,
			Output: fn(inputSlice...),
		}
	}

	return table
}

// String returns a formatted string representation of the truth table.
// The output is formatted as a table with columns for each variable
// and the output, using 'T' for true and 'F' for false values.
//
// Example output:
//
//	A       B       Output
//	------------------
//	F       F       F
//	F       T       T
//	T       F       T
//	T       T       T
func (tt *TruthTable) String() string {
	if len(tt.Rows) == 0 {
		return "Empty truth table\n"
	}

	var builder strings.Builder

	// Header - variable names and output
	for _, variable := range tt.Variables {
		builder.WriteString(fmt.Sprintf("%-8s", variable))
	}
	builder.WriteString("Output\n")

	// Separator line
	totalWidth := len(tt.Variables)*8 + 6
	builder.WriteString(strings.Repeat("-", totalWidth))
	builder.WriteString("\n")

	// Data rows
	for _, row := range tt.Rows {
		// Input values
		for _, variable := range tt.Variables {
			value := row.Inputs[variable]
			if value {
				builder.WriteString("T       ")
			} else {
				builder.WriteString("F       ")
			}
		}

		// Output value
		if row.Output {
			builder.WriteString("T")
		} else {
			builder.WriteString("F")
		}
		builder.WriteString("\n")
	}

	return builder.String()
}
