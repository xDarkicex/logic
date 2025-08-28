package core

import "fmt"

// TruthTable represents a generic truth table
type TruthTable interface {
	Variables() []string
	Rows() []TruthTableRow
	String() string
}

// TruthTableRow represents a single row
type TruthTableRow interface {
	Inputs() map[string]interface{}
	Output() interface{}
}

// BasicEvaluationContext implements EvaluationContext
type BasicEvaluationContext struct {
	vars map[string]interface{}
}

func NewEvaluationContext() *BasicEvaluationContext {
	return &BasicEvaluationContext{
		vars: make(map[string]interface{}),
	}
}

func (ctx *BasicEvaluationContext) Get(variable string) (interface{}, bool) {
	val, exists := ctx.vars[variable]
	return val, exists
}

func (ctx *BasicEvaluationContext) Set(variable string, value interface{}) {
	ctx.vars[variable] = value
}

func (ctx *BasicEvaluationContext) Clone() EvaluationContext {
	newCtx := NewEvaluationContext()
	for k, v := range ctx.vars {
		newCtx.vars[k] = v
	}
	return newCtx
}

func (ctx *BasicEvaluationContext) Variables() []string {
	vars := make([]string, 0, len(ctx.vars))
	for k := range ctx.vars {
		vars = append(vars, k)
	}
	return vars
}

// LogicEngine implements Engine
type LogicEngine struct {
	systems map[string]LogicSystem
}

func NewLogicEngine() *LogicEngine {
	return &LogicEngine{
		systems: make(map[string]LogicSystem),
	}
}

func (e *LogicEngine) RegisterSystem(name string, system LogicSystem) {
	e.systems[name] = system
}

func (e *LogicEngine) GetSystem(name string) (LogicSystem, bool) {
	system, exists := e.systems[name]
	return system, exists
}

func (e *LogicEngine) ListSystems() []string {
	names := make([]string, 0, len(e.systems))
	for name := range e.systems {
		names = append(names, name)
	}
	return names
}

func (e *LogicEngine) ConvertBetween(expr string, fromSystem, toSystem string) (string, error) {
	// Placeholder - implement in future phases
	return "", fmt.Errorf("conversion not yet implemented")
}
