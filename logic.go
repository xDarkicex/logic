package logic

import (
	"github.com/xDarkicex/logic/classical"
	"github.com/xDarkicex/logic/core"
	"github.com/xDarkicex/logic/sat"
)

// DefaultEngine provides backwards compatibility
var DefaultEngine *core.LogicEngine

func init() {
	DefaultEngine = core.NewLogicEngine()
	DefaultEngine.RegisterSystem("classical", classical.NewClassicalSystem())
	DefaultEngine.RegisterSystem("sat", sat.NewSATSystem())
}

// Backwards compatibility functions
func And(inputs ...bool) bool {
	return classical.And(inputs...)
}

func Or(inputs ...bool) bool {
	return classical.Or(inputs...)
}

func Xor(inputs ...bool) bool {
	return classical.Xor(inputs...)
}

func Not(input bool) bool {
	return classical.Not(input)
}

func Nand(inputs ...bool) bool {
	return classical.Nand(inputs...)
}

func Nor(inputs ...bool) bool {
	return classical.Nor(inputs...)
}

func Xnor(inputs ...bool) bool {
	return classical.Xnor(inputs...)
}

func Implies(a, b bool) bool {
	return classical.Implies(a, b)
}

func Iff(a, b bool) bool {
	return classical.Iff(a, b)
}

// Add SAT convenience functions
var SolveSAT = func(expr string) (*sat.SolverResult, error) {
	system, exists := DefaultEngine.GetSystem("sat")
	if !exists {
		return nil, core.NewLogicError("", "SolveSAT", "SAT system not available")
	}

	satSys := system.(*sat.SATSystemImpl)
	cnf, err := satSys.ConvertToCNF(expr)
	if err != nil {
		return nil, err
	}

	return satSys.Solve(cnf), nil
}

// Type aliases for backwards compatibility
type BoolVector = classical.BoolVector
type BitwiseInt = classical.BitwiseInt
type TruthTable = classical.TruthTable
type TruthTableRow = classical.TruthTableRow
type Circuit = classical.Circuit
type Gate = classical.Gate
type Evaluator = classical.Evaluator

// Function aliases
var NewBoolVector = classical.NewBoolVector
var NewBitwiseInt = classical.NewBitwiseInt
var NewCircuit = classical.NewCircuit
var Eval = classical.Eval
var GenerateTruthTable = classical.GenerateTruthTable
var EvaluateExpression = classical.EvaluateExpression
var ValidateExpression = classical.ValidateExpression
var Tautology = classical.Tautology
var Contradiction = classical.Contradiction
var Contingency = classical.Contingency

// Gate aliases
type AndGate = classical.AndGate
type OrGate = classical.OrGate
type NotGate = classical.NotGate
type XorGate = classical.XorGate
type XnorGate = classical.XnorGate
type NandGate = classical.NandGate
type NorGate = classical.NorGate

var DeMorganLaw = classical.DeMorganLaw
var DistributiveLaw = classical.DistributiveLaw

type LogicError = core.LogicError
