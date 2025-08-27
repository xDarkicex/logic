package logic

// Gate represents a logical gate interface that can evaluate boolean inputs.
// All logical gates implement this interface, providing a uniform way to
// work with different types of logic gates in circuits and simulations.
type Gate interface {
	// Evaluate processes the input values and returns the gate's output.
	// The number and meaning of inputs depends on the specific gate type.
	Evaluate(inputs ...bool) bool

	// String returns a string representation of the gate type.
	String() string
}

// AndGate implements a logical AND gate.
// Returns true only when all inputs are true.
type AndGate struct{}

// Evaluate implements the Gate interface for AndGate.
// Returns true if all inputs are true, false otherwise.
//
// Example:
//
//	gate := AndGate{}
//	result := gate.Evaluate(true, true)  // true
//	result = gate.Evaluate(true, false)  // false
func (g AndGate) Evaluate(inputs ...bool) bool {
	return And(inputs...)
}

// String returns the name of the gate.
func (g AndGate) String() string {
	return "AND"
}

// OrGate implements a logical OR gate.
// Returns true when at least one input is true.
type OrGate struct{}

// Evaluate implements the Gate interface for OrGate.
// Returns true if any input is true, false if all inputs are false.
//
// Example:
//
//	gate := OrGate{}
//	result := gate.Evaluate(false, true)  // true
//	result = gate.Evaluate(false, false)  // false
func (g OrGate) Evaluate(inputs ...bool) bool {
	return Or(inputs...)
}

// String returns the name of the gate.
func (g OrGate) String() string {
	return "OR"
}

// NotGate implements a logical NOT gate (inverter).
// Returns the opposite of the single input value.
type NotGate struct{}

// Evaluate implements the Gate interface for NotGate.
// Expects exactly one input. Returns false if input is true, true if input is false.
// Returns false if no inputs or multiple inputs are provided.
//
// Example:
//
//	gate := NotGate{}
//	result := gate.Evaluate(true)   // false
//	result = gate.Evaluate(false)   // true
func (g NotGate) Evaluate(inputs ...bool) bool {
	if len(inputs) != 1 {
		return false
	}
	return Not(inputs[0])
}

// String returns the name of the gate.
func (g NotGate) String() string {
	return "NOT"
}

// XorGate implements a logical XOR (exclusive OR) gate.
// Returns true when an odd number of inputs are true.
type XorGate struct{}

// Evaluate implements the Gate interface for XorGate.
// Returns true if an odd number of inputs are true.
//
// Example:
//
//	gate := XorGate{}
//	result := gate.Evaluate(true, false)   // true
//	result = gate.Evaluate(true, true)     // false
//	result = gate.Evaluate(true, true, true) // true
func (g XorGate) Evaluate(inputs ...bool) bool {
	return Xor(inputs...)
}

// String returns the name of the gate.
func (g XorGate) String() string {
	return "XOR"
}

// NandGate implements a logical NAND (NOT AND) gate.
// Returns the opposite of an AND gate - false only when all inputs are true.
type NandGate struct{}

// Evaluate implements the Gate interface for NandGate.
// Returns false if all inputs are true, true otherwise.
//
// Example:
//
//	gate := NandGate{}
//	result := gate.Evaluate(true, true)   // false
//	result = gate.Evaluate(true, false)   // true
func (g NandGate) Evaluate(inputs ...bool) bool {
	return Nand(inputs...)
}

// String returns the name of the gate.
func (g NandGate) String() string {
	return "NAND"
}

// NorGate implements a logical NOR (NOT OR) gate.
// Returns the opposite of an OR gate - true only when all inputs are false.
type NorGate struct{}

// Evaluate implements the Gate interface for NorGate.
// Returns true if all inputs are false, false otherwise.
//
// Example:
//
//	gate := NorGate{}
//	result := gate.Evaluate(false, false)  // true
//	result = gate.Evaluate(false, true)    // false
func (g NorGate) Evaluate(inputs ...bool) bool {
	return Nor(inputs...)
}

// String returns the name of the gate.
func (g NorGate) String() string {
	return "NOR"
}

// Circuit represents a collection of logic gates that can be simulated together.
// It maintains a list of input variables and gates, allowing for complex
// logic circuit simulation.
type Circuit struct {
	inputs []string
	gates  []Gate
}

// NewCircuit creates a new circuit with the specified input variable names.
// The inputs define the external inputs to the circuit that can be varied
// during simulation.
//
// Example:
//
//	circuit := NewCircuit([]string{"A", "B", "C"})
func NewCircuit(inputs []string) *Circuit {
	return &Circuit{
		inputs: make([]string, len(inputs)),
		gates:  make([]Gate, 0),
	}
}

// AddGate adds a gate to the circuit.
// Gates are evaluated in the order they are added. For complex circuits,
// the order of gate addition may affect the simulation results.
//
// Example:
//
//	circuit := NewCircuit([]string{"A", "B"})
//	circuit.AddGate(AndGate{})
//	circuit.AddGate(NotGate{})
func (c *Circuit) AddGate(gate Gate) {
	c.gates = append(c.gates, gate)
}

// Simulate runs the circuit with the given input values.
// The inputs map should contain values for all input variables defined
// when creating the circuit. Returns an error if no gates are present.
//
// Note: This is a simplified simulation that evaluates only the first gate
// with all inputs. A more sophisticated implementation would handle
// gate interconnections and signal propagation.
//
// Example:
//
//	circuit := NewCircuit([]string{"A", "B"})
//	circuit.AddGate(AndGate{})
//	inputs := map[string]bool{"A": true, "B": false}
//	result, err := circuit.Simulate(inputs) // false, nil
func (c *Circuit) Simulate(inputs map[string]bool) (bool, error) {
	if len(c.gates) == 0 {
		return false, NewLogicError("Circuit.Simulate", "no gates in circuit")
	}

	// Simple simulation - evaluate first gate with all inputs
	// In a real implementation, this would be more sophisticated
	inputValues := make([]bool, 0, len(inputs))
	for _, inputName := range c.inputs {
		if val, exists := inputs[inputName]; exists {
			inputValues = append(inputValues, val)
		}
	}

	return c.gates[0].Evaluate(inputValues...), nil
}
