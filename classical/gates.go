package classical

import (
	"fmt"
	"strings"

	"github.com/xDarkicex/logic/core"
)

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

// XnorGate implements a logical XNOR (exclusive NOR) gate.
// Returns true when an even number of inputs are true (opposite of XOR).
type XnorGate struct{}

// Evaluate implements the Gate interface for XnorGate.
// Returns true if an even number of inputs are true.
//
// Example:
//
//	gate := XnorGate{}
//	result := gate.Evaluate(true, false)   // false
//	result = gate.Evaluate(true, true)     // true
//	result = gate.Evaluate(false, false)   // true
func (g XnorGate) Evaluate(inputs ...bool) bool {
	return !Xor(inputs...)
}

// String returns the name of the gate.
func (g XnorGate) String() string {
	return "XNOR"
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

// CircuitNode represents a single logic gate node in a circuit.
// Each node has a unique ID, contains a gate, and can be connected to other nodes.
type CircuitNode struct {
	// ID is the unique identifier for this node in the circuit
	ID string

	// Gate is the logical operation this node performs
	Gate Gate

	// Inputs contains references to other node IDs or input variable names
	// that provide inputs to this node's gate
	Inputs []string

	// Value caches the computed result, nil if not yet computed
	Value *bool
}

// Circuit represents a complete logic circuit with interconnected gates.
// It supports multiple inputs, multiple outputs, and proper signal propagation
// through a network of connected logic gates.
type Circuit struct {
	// InputVars defines the external input variables to the circuit
	InputVars []string

	// Nodes contains all the logic gate nodes indexed by their ID
	Nodes map[string]*CircuitNode

	// Outputs specifies which node IDs represent circuit outputs
	Outputs []string

	// topology contains topologically sorted node IDs for correct evaluation order
	topology []string

	// topologyValid tracks if the current topology is valid
	topologyValid bool
}

// NewCircuit creates a new circuit with the specified input variables.
// Input variables are the external signals that can be varied during simulation.
//
// Example:
//
//	circuit := NewCircuit([]string{"A", "B", "C"})
func NewCircuit(inputs []string) *Circuit {
	inputVars := make([]string, len(inputs))
	copy(inputVars, inputs) // Actually copy the input values
	return &Circuit{
		InputVars:     inputVars,
		Nodes:         make(map[string]*CircuitNode),
		Outputs:       make([]string, 0),
		topology:      make([]string, 0),
		topologyValid: false,
	}
}

// AddNode adds a logic gate node to the circuit.
// The node ID must be unique, and input references should either be
// input variable names or IDs of previously added nodes.
//
// Example:
//
//	circuit.AddNode("gate1", AndGate{}, []string{"A", "B"})
//	circuit.AddNode("gate2", OrGate{}, []string{"gate1", "C"})
func (c *Circuit) AddNode(nodeID string, gate Gate, inputs []string) error {
	if _, exists := c.Nodes[nodeID]; exists {
		return core.NewLogicError("classical", "Circuit.AddNode", fmt.Sprintf("node ID '%s' already exists", nodeID))
	}

	c.Nodes[nodeID] = &CircuitNode{
		ID:     nodeID,
		Gate:   gate,
		Inputs: make([]string, len(inputs)),
		Value:  nil,
	}
	copy(c.Nodes[nodeID].Inputs, inputs)

	// Invalidate topology since circuit structure changed
	c.topologyValid = false

	return nil
}

// SetOutputs specifies which nodes represent the circuit's outputs.
// These are the nodes whose values will be returned by Simulate().
//
// Example:
//
//	circuit.SetOutputs([]string{"gate2", "gate3"})
func (c *Circuit) SetOutputs(outputs []string) error {
	// Validate that all output nodes exist
	for _, outputID := range outputs {
		if _, exists := c.Nodes[outputID]; !exists {
			return core.NewLogicError("classical", "Circuit.SetOutputs", fmt.Sprintf("output node '%s' does not exist", outputID))
		}
	}

	c.Outputs = make([]string, len(outputs))
	copy(c.Outputs, outputs)
	return nil
}

// buildTopology creates a topological ordering of nodes for evaluation.
// This ensures that nodes are evaluated in the correct dependency order.
func (c *Circuit) buildTopology() error {
	if c.topologyValid {
		return nil
	}

	visited := make(map[string]bool)
	tempMark := make(map[string]bool)
	topology := make([]string, 0, len(c.Nodes))

	var visit func(string) error
	visit = func(nodeID string) error {
		if tempMark[nodeID] {
			return core.NewLogicError("classical", "Circuit.buildTopology", "circular dependency detected")
		}
		if visited[nodeID] {
			return nil
		}

		tempMark[nodeID] = true

		node := c.Nodes[nodeID]
		for _, inputRef := range node.Inputs {
			// Only visit if it's another node (not an input variable)
			if _, isNode := c.Nodes[inputRef]; isNode {
				if err := visit(inputRef); err != nil {
					return err
				}
			}
		}

		tempMark[nodeID] = false
		visited[nodeID] = true
		topology = append(topology, nodeID)

		return nil
	}

	// Visit all nodes
	for nodeID := range c.Nodes {
		if !visited[nodeID] {
			if err := visit(nodeID); err != nil {
				return err
			}
		}
	}

	c.topology = topology
	c.topologyValid = true
	return nil
}

// Simulate runs the circuit with the given input values and returns the output values.
// The inputs map should contain values for all input variables defined when creating the circuit.
// Returns a map of output node IDs to their computed boolean values.
//
// Example:
//
//	inputs := map[string]bool{"A": true, "B": false, "C": true}
//	outputs, err := circuit.Simulate(inputs)
//	if err == nil {
//		fmt.Printf("Output gate2: %v\n", outputs["gate2"])
//	}
func (c *Circuit) Simulate(inputs map[string]bool) (map[string]bool, error) {
	// Validate all required inputs are provided
	for _, inputVar := range c.InputVars {
		if _, exists := inputs[inputVar]; !exists {
			return nil, core.NewLogicError("classical", "Circuit.Simulate", fmt.Sprintf("missing input value for '%s'", inputVar))
		}
	}

	// Build topology if needed
	if err := c.buildTopology(); err != nil {
		return nil, err
	}

	// Reset all cached values
	for _, node := range c.Nodes {
		node.Value = nil
	}

	// Evaluate nodes in topological order
	for _, nodeID := range c.topology {
		node := c.Nodes[nodeID]
		inputValues := make([]bool, len(node.Inputs))

		// Resolve input values
		for i, inputRef := range node.Inputs {
			if val, exists := inputs[inputRef]; exists {
				// Input variable
				inputValues[i] = val
			} else if refNode, exists := c.Nodes[inputRef]; exists && refNode.Value != nil {
				// Node output
				inputValues[i] = *refNode.Value
			} else {
				return nil, core.NewLogicError("classical", "Circuit.Simulate",
					fmt.Sprintf("unresolved input '%s' for node '%s'", inputRef, nodeID))
			}
		}

		// Evaluate gate and cache result
		result := node.Gate.Evaluate(inputValues...)
		node.Value = &result
	}

	// Collect output values
	results := make(map[string]bool)
	for _, outputID := range c.Outputs {
		if node, exists := c.Nodes[outputID]; exists && node.Value != nil {
			results[outputID] = *node.Value
		} else {
			return nil, core.NewLogicError("classical", "Circuit.Simulate", fmt.Sprintf("output node '%s' not evaluated", outputID))
		}
	}

	return results, nil
}

// GetNodeValue returns the cached value of a specific node after simulation.
// Returns an error if the node doesn't exist or hasn't been evaluated.
func (c *Circuit) GetNodeValue(nodeID string) (bool, error) {
	node, exists := c.Nodes[nodeID]
	if !exists {
		return false, core.NewLogicError("classical", "Circuit.GetNodeValue", fmt.Sprintf("node '%s' does not exist", nodeID))
	}
	if node.Value == nil {
		return false, core.NewLogicError("classical", "Circuit.GetNodeValue", fmt.Sprintf("node '%s' has not been evaluated", nodeID))
	}
	return *node.Value, nil
}

// String returns a string representation of the circuit structure.
func (c *Circuit) String() string {
	var builder strings.Builder

	builder.WriteString("Circuit:\n")
	builder.WriteString(fmt.Sprintf("  Inputs: %v\n", c.InputVars))
	builder.WriteString(fmt.Sprintf("  Outputs: %v\n", c.Outputs))
	builder.WriteString("  Nodes:\n")

	for nodeID, node := range c.Nodes {
		valueStr := "nil"
		if node.Value != nil {
			valueStr = fmt.Sprintf("%v", *node.Value)
		}
		builder.WriteString(fmt.Sprintf("    %s: %s(%v) = %s\n",
			nodeID, node.Gate.String(), node.Inputs, valueStr))
	}

	return builder.String()
}
