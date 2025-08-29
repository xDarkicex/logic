package sat

import (
	"fmt"

	"github.com/xDarkicex/logic/classical"
	"github.com/xDarkicex/logic/core"
)

// CNFConverter converts logical expressions to CNF using Tseitin transformation
type CNFConverter struct {
	nextAuxVar int // Counter for auxiliary variables
	cnf        *CNF
}

// NewCNFConverter creates a new CNF converter
func NewCNFConverter() *CNFConverter {
	return &CNFConverter{
		nextAuxVar: 1,
		cnf:        NewCNF(),
	}
}

// ConvertExpression converts a logical expression to CNF
func (c *CNFConverter) ConvertExpression(expr string) (*CNF, error) {
	// Parse expression using existing parser
	ast, err := classical.ParseExpression(expr)
	if err != nil {
		return nil, core.NewLogicError("sat", "CNFConverter.ConvertExpression",
			fmt.Sprintf("failed to parse expression: %v", err))
	}

	c.cnf = NewCNF()
	c.nextAuxVar = 1

	// Convert AST to CNF using Tseitin transformation
	rootVar, err := c.tseitinTransform(ast)
	if err != nil {
		return nil, err
	}

	// Add unit clause to ensure root is true
	rootLiteral := Literal{Variable: rootVar, Negated: false}
	c.cnf.AddClause(NewClause(rootLiteral))
	return c.cnf, nil
}

// ConvertAST converts AST node directly to CNF
func (c *CNFConverter) ConvertAST(node *classical.ASTNode) (*CNF, error) {
	c.cnf = NewCNF()
	c.nextAuxVar = 1

	rootVar, err := c.tseitinTransform(node)
	if err != nil {
		return nil, err
	}

	// Add unit clause to ensure root is true
	rootLiteral := Literal{Variable: rootVar, Negated: false}
	c.cnf.AddClause(NewClause(rootLiteral))
	return c.cnf, nil
}

// tseitinTransform performs Tseitin transformation
// Returns the variable representing this subexpression
func (c *CNFConverter) tseitinTransform(node *classical.ASTNode) (string, error) {
	switch node.Type {
	case classical.NodeVariable:
		return node.Value, nil

	case classical.NodeConstant:
		// Create auxiliary variable for constant
		auxVar := c.getNextAuxVar()
		if node.Value == "true" || node.Value == "1" || node.Value == "T" {
			// Add unit clause: auxVar
			c.cnf.AddClause(NewClause(Literal{Variable: auxVar, Negated: false}))
		} else {
			// Add unit clause: ¬auxVar
			c.cnf.AddClause(NewClause(Literal{Variable: auxVar, Negated: true}))
		}
		return auxVar, nil

	case classical.NodeNot:
		if len(node.Children) != 1 {
			return "", core.NewLogicError("sat", "CNFConverter.tseitinTransform",
				"NOT node must have exactly one child")
		}
		// Index the single child
		childVar, err := c.tseitinTransform(node.Children[0])
		if err != nil {
			return "", err
		}

		auxVar := c.getNextAuxVar()
		// auxVar ↔ ¬childVar
		// (auxVar ∨ childVar) ∧ (¬auxVar ∨ ¬childVar)
		c.cnf.AddClause(NewClause(
			Literal{Variable: auxVar, Negated: false},
			Literal{Variable: childVar, Negated: false},
		))
		c.cnf.AddClause(NewClause(
			Literal{Variable: auxVar, Negated: true},
			Literal{Variable: childVar, Negated: true},
		))
		return auxVar, nil

	case classical.NodeAnd:
		return c.transformAnd(node)

	case classical.NodeOr:
		return c.transformOr(node)

	case classical.NodeXor:
		return c.transformXor(node)

	case classical.NodeNand:
		return c.transformNand(node)

	case classical.NodeNor:
		return c.transformNor(node)

	case classical.NodeImplies:
		return c.transformImplies(node)

	case classical.NodeIff:
		return c.transformIff(node)

	default:
		return "", core.NewLogicError("sat", "CNFConverter.tseitinTransform",
			fmt.Sprintf("unsupported node type: %v", node.Type))
	}
}

// transformAnd handles AND operations: p ↔ (∧ children)
// CNF: ∧i (¬p ∨ vi) ∧ (p ∨ ¬v1 ∨ ... ∨ ¬vn)
func (c *CNFConverter) transformAnd(node *classical.ASTNode) (string, error) {
	if len(node.Children) == 0 {
		return "", core.NewLogicError("sat", "CNFConverter.transformAnd",
			"AND node must have children")
	}

	// Transform children
	childVars := make([]string, len(node.Children))
	for i, child := range node.Children {
		var err error
		childVars[i], err = c.tseitinTransform(child)
		if err != nil {
			return "", err
		}
	}

	auxVar := c.getNextAuxVar()

	// (¬auxVar ∨ childi) for each child
	for _, childVar := range childVars {
		c.cnf.AddClause(NewClause(
			Literal{Variable: auxVar, Negated: true},
			Literal{Variable: childVar, Negated: false},
		))
	}

	// (auxVar ∨ ¬child1 ∨ ... ∨ ¬childN)
	literals := make([]Literal, 0, len(childVars)+1)
	literals = append(literals, Literal{Variable: auxVar, Negated: false})
	for _, childVar := range childVars {
		literals = append(literals, Literal{Variable: childVar, Negated: true})
	}
	c.cnf.AddClause(NewClause(literals...))

	return auxVar, nil
}

// transformOr handles OR operations: p ↔ (∨ children)
// CNF: (¬p ∨ v1 ∨ ... ∨ vn) ∧ ∧i (p ∨ ¬vi)
func (c *CNFConverter) transformOr(node *classical.ASTNode) (string, error) {
	if len(node.Children) == 0 {
		return "", core.NewLogicError("sat", "CNFConverter.transformOr",
			"OR node must have children")
	}

	// Transform children
	childVars := make([]string, len(node.Children))
	for i, child := range node.Children {
		var err error
		childVars[i], err = c.tseitinTransform(child)
		if err != nil {
			return "", err
		}
	}

	auxVar := c.getNextAuxVar()

	// First clause: (¬auxVar ∨ child1 ∨ ... ∨ childN)
	lits := make([]Literal, 0, len(childVars)+1)
	lits = append(lits, Literal{Variable: auxVar, Negated: true})
	for _, childVar := range childVars {
		lits = append(lits, Literal{Variable: childVar, Negated: false})
	}
	c.cnf.AddClause(NewClause(lits...))

	// Per-child clauses: (auxVar ∨ ¬childi)
	for _, childVar := range childVars {
		c.cnf.AddClause(NewClause(
			Literal{Variable: auxVar, Negated: false},
			Literal{Variable: childVar, Negated: true},
		))
	}

	return auxVar, nil
}

// transformXor handles XOR operations: p ↔ (a ⊕ b)
// CNF: (¬p ∨ ¬a ∨ ¬b) ∧ (¬p ∨ a ∨ b) ∧ (p ∨ ¬a ∨ b) ∧ (p ∨ a ∨ ¬b)
func (c *CNFConverter) transformXor(node *classical.ASTNode) (string, error) {
	if len(node.Children) != 2 {
		return "", core.NewLogicError("sat", "CNFConverter.transformXor",
			"XOR node must have exactly two children")
	}

	// Index children
	child1Var, err := c.tseitinTransform(node.Children[0])
	if err != nil {
		return "", err
	}

	child2Var, err := c.tseitinTransform(node.Children[1])
	if err != nil {
		return "", err
	}

	auxVar := c.getNextAuxVar()

	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: true},
		Literal{Variable: child1Var, Negated: true},
		Literal{Variable: child2Var, Negated: true},
	))
	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: true},
		Literal{Variable: child1Var, Negated: false},
		Literal{Variable: child2Var, Negated: false},
	))
	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: false},
		Literal{Variable: child1Var, Negated: true},
		Literal{Variable: child2Var, Negated: false},
	))
	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: false},
		Literal{Variable: child1Var, Negated: false},
		Literal{Variable: child2Var, Negated: true},
	))

	return auxVar, nil
}

// transformImplies handles IMPLIES operations: p ↔ (a → b) = p ↔ (¬a ∨ b)
// CNF: (¬p ∨ ¬a ∨ b) ∧ (p ∨ a) ∧ (p ∨ ¬b)
func (c *CNFConverter) transformImplies(node *classical.ASTNode) (string, error) {
	if len(node.Children) != 2 {
		return "", core.NewLogicError("sat", "CNFConverter.transformImplies",
			"IMPLIES node must have exactly two children")
	}

	// Index children
	child1Var, err := c.tseitinTransform(node.Children[0])
	if err != nil {
		return "", err
	}

	child2Var, err := c.tseitinTransform(node.Children[1])
	if err != nil {
		return "", err
	}

	auxVar := c.getNextAuxVar()

	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: true},
		Literal{Variable: child1Var, Negated: true},
		Literal{Variable: child2Var, Negated: false},
	))
	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: false},
		Literal{Variable: child1Var, Negated: false},
	))
	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: false},
		Literal{Variable: child2Var, Negated: true},
	))

	return auxVar, nil
}

// transformIff handles IFF operations: p ↔ (a ↔ b)
// CNF: (p ∨ ¬a ∨ ¬b) ∧ (p ∨ a ∨ b) ∧ (¬p ∨ ¬a ∨ b) ∧ (¬p ∨ a ∨ ¬b)
func (c *CNFConverter) transformIff(node *classical.ASTNode) (string, error) {
	if len(node.Children) != 2 {
		return "", core.NewLogicError("sat", "CNFConverter.transformIff",
			"IFF node must have exactly two children")
	}

	// Index children
	child1Var, err := c.tseitinTransform(node.Children[0])
	if err != nil {
		return "", err
	}

	child2Var, err := c.tseitinTransform(node.Children[1])
	if err != nil {
		return "", err
	}

	auxVar := c.getNextAuxVar()

	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: false},
		Literal{Variable: child1Var, Negated: true},
		Literal{Variable: child2Var, Negated: true},
	))
	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: false},
		Literal{Variable: child1Var, Negated: false},
		Literal{Variable: child2Var, Negated: false},
	))
	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: true},
		Literal{Variable: child1Var, Negated: true},
		Literal{Variable: child2Var, Negated: false},
	))
	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: true},
		Literal{Variable: child1Var, Negated: false},
		Literal{Variable: child2Var, Negated: true},
	))

	return auxVar, nil
}

// transformNand: NAND(A,B,...) = ¬(A ∧ B ∧ ...)
func (c *CNFConverter) transformNand(node *classical.ASTNode) (string, error) {
	andVar, err := c.transformAnd(node)
	if err != nil {
		return "", err
	}

	auxVar := c.getNextAuxVar()
	// auxVar ↔ ¬andVar
	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: false},
		Literal{Variable: andVar, Negated: false},
	))
	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: true},
		Literal{Variable: andVar, Negated: true},
	))
	return auxVar, nil
}

// transformNor: NOR(A,B,...) = ¬(A ∨ B ∨ ...)
func (c *CNFConverter) transformNor(node *classical.ASTNode) (string, error) {
	orVar, err := c.transformOr(node)
	if err != nil {
		return "", err
	}

	auxVar := c.getNextAuxVar()
	// auxVar ↔ ¬orVar
	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: false},
		Literal{Variable: orVar, Negated: false},
	))
	c.cnf.AddClause(NewClause(
		Literal{Variable: auxVar, Negated: true},
		Literal{Variable: orVar, Negated: true},
	))
	return auxVar, nil
}

// getNextAuxVar generates next auxiliary variable name
func (c *CNFConverter) getNextAuxVar() string {
	auxVar := fmt.Sprintf("__aux_%d", c.nextAuxVar)
	c.nextAuxVar++
	return auxVar
}

// Add to CNFConverter to handle XOR more efficiently
func (c *CNFConverter) ConvertExpressionExtended(expr string) (*ExtendedCNF, error) {
	ast, err := classical.ParseExpression(expr)
	if err != nil {
		return nil, core.NewLogicError("sat", "CNFConverter.ConvertExpressionExtended",
			fmt.Sprintf("failed to parse expression: %v", err))
	}

	ecnf := NewExtendedCNF()
	c.cnf = ecnf.CNF
	c.nextAuxVar = 1

	// Convert AST with XOR detection
	rootVar, err := c.tseitinTransformExtended(ast, ecnf)
	if err != nil {
		return nil, err
	}

	// Add unit clause to ensure root is true
	rootLiteral := Literal{Variable: rootVar, Negated: false}
	ecnf.AddClause(NewClause(rootLiteral))

	return ecnf, nil
}

// Enhanced transformation that can generate XOR clauses directly
func (c *CNFConverter) tseitinTransformExtended(node *classical.ASTNode, ecnf *ExtendedCNF) (string, error) {
	switch node.Type {
	case classical.NodeXor:
		// Generate XOR clause directly instead of exponential expansion
		if len(node.Children) >= 2 && len(node.Children) <= 10 { // Reasonable XOR size
			return c.transformXorDirect(node, ecnf)
		}
		// Fall back to regular transformation for large XORs
		return c.transformXor(node)

	default:
		// Use existing transformation methods
		return c.tseitinTransform(node)
	}
}

// transformXorDirect creates XOR clause directly
func (c *CNFConverter) transformXorDirect(node *classical.ASTNode, ecnf *ExtendedCNF) (string, error) {
	if len(node.Children) < 2 {
		return "", core.NewLogicError("sat", "CNFConverter.transformXorDirect",
			"XOR node must have at least two children")
	}

	// Transform children
	childVars := make([]string, len(node.Children))
	for i, child := range node.Children {
		var err error
		childVars[i], err = c.tseitinTransformExtended(child, ecnf)
		if err != nil {
			return "", err
		}
	}

	auxVar := c.getNextAuxVar()

	// Create XOR clause: auxVar ⊕ child1 ⊕ child2 ⊕ ... = 1 (odd parity)
	xorVars := make([]string, 0, len(childVars)+1)
	xorVars = append(xorVars, auxVar)
	xorVars = append(xorVars, childVars...)
	xorClause := NewXORClause(xorVars, true) // odd parity
	ecnf.AddXORClause(xorClause)

	return auxVar, nil
}
