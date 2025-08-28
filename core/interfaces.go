package core

// LogicSystem represents any logical reasoning system
type LogicSystem interface {
	Name() string
	Evaluate(expr string, ctx EvaluationContext) (interface{}, error)
	Validate(expr string) error
	SupportedOperators() []string
}

// EvaluationContext provides variable bindings for evaluation
type EvaluationContext interface {
	Get(variable string) (interface{}, bool)
	Set(variable string, value interface{})
	Clone() EvaluationContext
	Variables() []string
}

// TruthTableGenerator can generate truth tables
type TruthTableGenerator interface {
	GenerateTable(variables []string, fn func(...interface{}) interface{}) (TruthTable, error)
}

// CircuitSimulator can simulate logical circuits
type CircuitSimulator interface {
	AddNode(nodeID string, gate Gate, inputs []string) error
	SetOutputs(outputs []string) error
	Simulate(inputs map[string]interface{}) (map[string]interface{}, error)
}

// Gate represents a logical gate
type Gate interface {
	Evaluate(inputs ...interface{}) interface{}
	String() string
	InputCount() int // -1 for variadic
	OutputCount() int
}

// Engine coordinates multiple logic systems
type Engine interface {
	RegisterSystem(name string, system LogicSystem)
	GetSystem(name string) (LogicSystem, bool)
	ListSystems() []string
	ConvertBetween(expr string, fromSystem, toSystem string) (string, error)
}
