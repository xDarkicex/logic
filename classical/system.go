package classical

import (
	"github.com/xDarkicex/logic/core"
)

// ClassicalSystem implements core.LogicSystem
type ClassicalSystem struct{}

func NewClassicalSystem() *ClassicalSystem {
	return &ClassicalSystem{}
}

func (s *ClassicalSystem) Name() string {
	return "classical"
}

func (s *ClassicalSystem) Evaluate(expr string, ctx core.EvaluationContext) (interface{}, error) {
	// Convert core.EvaluationContext to map[string]bool
	vars := make(map[string]bool)
	for _, varName := range ctx.Variables() {
		if val, exists := ctx.Get(varName); exists {
			if boolVal, ok := val.(bool); ok {
				vars[varName] = boolVal
			}
		}
	}

	return EvaluateExpression(expr, vars)
}

func (s *ClassicalSystem) Validate(expr string) error {
	return ValidateExpression(expr)
}

func (s *ClassicalSystem) SupportedOperators() []string {
	return []string{"&", "∧", "|", "∨", "^", "⊕", "!", "¬", "->", "→", "<->", "↔", "and", "or", "xor", "not", "nand", "nor", "implies", "iff"}
}

// Ensure ClassicalSystem implements interfaces
var _ core.LogicSystem = (*ClassicalSystem)(nil)
var _ core.TruthTableGenerator = (*ClassicalSystem)(nil)

func (s *ClassicalSystem) GenerateTable(variables []string, fn func(...interface{}) interface{}) (core.TruthTable, error) {
	// Convert interface{} function to bool function
	boolFn := func(inputs ...bool) bool {
		interfaceInputs := make([]interface{}, len(inputs))
		for i, input := range inputs {
			interfaceInputs[i] = input
		}
		result := fn(interfaceInputs...)
		if boolResult, ok := result.(bool); ok {
			return boolResult
		}
		return false
	}

	classicalTable := GenerateTruthTable(variables, boolFn)
	return &TruthTableAdapter{table: classicalTable}, nil
}

// TruthTableAdapter adapts classical TruthTable to core.TruthTable
type TruthTableAdapter struct {
	table *TruthTable
}

func (tta *TruthTableAdapter) Variables() []string {
	return tta.table.Variables
}

func (tta *TruthTableAdapter) Rows() []core.TruthTableRow {
	rows := make([]core.TruthTableRow, len(tta.table.Rows))
	for i, row := range tta.table.Rows {
		rows[i] = &TruthTableRowAdapter{row: &row}
	}
	return rows
}

func (tta *TruthTableAdapter) String() string {
	return tta.table.String()
}

// TruthTableRowAdapter adapts classical TruthTableRow to core.TruthTableRow
type TruthTableRowAdapter struct {
	row *TruthTableRow
}

func (ttra *TruthTableRowAdapter) Inputs() map[string]interface{} {
	inputs := make(map[string]interface{})
	for k, v := range ttra.row.Inputs {
		inputs[k] = v
	}
	return inputs
}

func (ttra *TruthTableRowAdapter) Output() interface{} {
	return ttra.row.Output
}
