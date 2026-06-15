package modal

import (
	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

// ModalLogicSystem is the unified runtime bridge for the modal logic package.
// It provides a high-level API for parsing, evaluating, and proving modal formulas,
// automatically routing to the correct subsystem (tableau prover, temporal evaluator, fuzzy bridge).
type ModalLogicSystem struct {
	pool    *memory.Pool
	arena   *memory.Arena
	prover  *Prover
	symbols map[string]fuzzy.VarID // small atom registry
	nextID  fuzzy.VarID
}

// NewModalLogicSystem creates a ModalLogicSystem backed by the given allocators.
func NewModalLogicSystem(pool *memory.Pool, arena *memory.Arena) *ModalLogicSystem {
	frame := NewFrame(pool, arena)
	frame.AddWorld() // world 0
	return &ModalLogicSystem{
		pool:    pool,
		arena:   arena,
		prover:  NewProver(pool, arena),
		symbols: make(map[string]fuzzy.VarID),
		nextID:  1,
	}
}

// proveOnFrame evaluates the formula on a fresh per-call frame (for tableaux).
func (s *ModalLogicSystem) proveOnFrame(formula Formula) (bool, *Model) {
	// Create a fresh frame for each proof — tableau creates worlds during expansion
	arena2, _ := memory.NewArena(1024 * 1024)
	defer arena2.Free()
	frame := NewFrame(s.pool, arena2)
	frame.AddWorld()
	return s.prover.ProveSatisfiable(formula, frame)
}

// Close releases internal resources.
func (s *ModalLogicSystem) Close() { s.prover.Close() }

// Name returns the system identifier.
func (s *ModalLogicSystem) Name() string { return "ModalLogicSystem" }

// Evaluate parses and checks whether a formula is satisfiable.
func (s *ModalLogicSystem) Evaluate(expression string) (bool, error) {
	formula, err := s.Parse(expression)
	if err != nil {
		return false, err
	}
	sat, _ := s.proveOnFrame(formula)
	return sat, nil
}

// IsValid parses and checks whether a formula is valid.
func (s *ModalLogicSystem) IsValid(expression string) (bool, error) {
	formula, err := s.Parse(expression)
	if err != nil {
		return false, err
	}
	neg := Not{Formula: formula}
	sat, _ := s.proveOnFrame(neg)
	return !sat, nil
}

// Validate checks whether the expression has valid syntax without evaluating it.
func (s *ModalLogicSystem) Validate(expression string) error {
	_, err := s.Parse(expression)
	return err
}

// Parse parses a modal formula string into a Formula.
func (s *ModalLogicSystem) Parse(expression string) (Formula, error) {
	lexer := NewLexer(expression, s.pool)
	tokens := lexer.Lex()
	parser := NewParser(tokens, expression)
	return parser.Parse()
}

// SupportedOperators returns the list of supported operators.
func (s *ModalLogicSystem) SupportedOperators() []string {
	return []string{
		"! (not)", "&& (and)", "|| (or)", "-> (implies)", "<-> (iff)",
		"[] (box)", "<> (diamond)", "X (next)", "U (until)",
		"G (globally/always)", "F (finally/eventually)",
	}
}

// ProveEntailsParsed checks entailment from parsed string premises.
func (s *ModalLogicSystem) ProveEntailsParsed(premiseExprs []string, conclusionExpr string) (bool, error) {
	premises := make([]Formula, len(premiseExprs))
	for i, expr := range premiseExprs {
		f, err := s.Parse(expr)
		if err != nil {
			return false, err
		}
		premises[i] = f
	}
	conc, err := s.Parse(conclusionExpr)
	if err != nil {
		return false, err
	}
	// Build frame for proof
	arena2, _ := memory.NewArena(1024 * 1024)
	defer arena2.Free()
	frame := NewFrame(s.pool, arena2)
	frame.AddWorld()
	return s.prover.ProveEntails(premises, conc, frame), nil
}
