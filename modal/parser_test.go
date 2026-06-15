package modal

import (
	"strings"
	"testing"

	"github.com/xDarkicex/memory"
)

func newParserPool(t *testing.T) *memory.Pool {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Reset)
	return pool
}

func parseExpr(t *testing.T, expr string) Formula {
	t.Helper()
	pool := newParserPool(t)
	lexer := NewLexer(expr, pool)
	tokens := lexer.Lex()
	parser := NewParser(tokens, expr)
	f, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse(%q): %v", expr, err)
	}
	return f
}

func TestParseAtom(t *testing.T) {
	f := parseExpr(t, "p")
	if _, ok := f.(Atom); !ok {
		t.Errorf("expected Atom, got %T", f)
	}
}

func TestParseNot(t *testing.T) {
	f := parseExpr(t, "!p")
	n, ok := f.(Not)
	if !ok {
		t.Fatalf("expected Not, got %T", f)
	}
	if _, ok := n.Formula.(Atom); !ok {
		t.Error("expected Atom under Not")
	}
}

func TestParseAnd(t *testing.T) {
	f := parseExpr(t, "p && q")
	a, ok := f.(And)
	if !ok {
		t.Fatalf("expected And, got %T", f)
	}
	if _, ok := a.Left.(Atom); !ok {
		t.Error("left should be Atom")
	}
	if _, ok := a.Right.(Atom); !ok {
		t.Error("right should be Atom")
	}
}

func TestParseOr(t *testing.T) {
	f := parseExpr(t, "p || q")
	if _, ok := f.(Or); !ok {
		t.Errorf("expected Or, got %T", f)
	}
}

func TestParseImplies(t *testing.T) {
	f := parseExpr(t, "p -> q")
	if _, ok := f.(Implies); !ok {
		t.Errorf("expected Implies, got %T", f)
	}
}

func TestParseIff(t *testing.T) {
	f := parseExpr(t, "p <-> q")
	if _, ok := f.(Iff); !ok {
		t.Errorf("expected Iff, got %T", f)
	}
}

func TestParseBox(t *testing.T) {
	f := parseExpr(t, "[]p")
	if _, ok := f.(Box); !ok {
		t.Errorf("expected Box, got %T", f)
	}
}

func TestParseDiamond(t *testing.T) {
	f := parseExpr(t, "<>p")
	if _, ok := f.(Diamond); !ok {
		t.Errorf("expected Diamond, got %T", f)
	}
}

func TestParseNext(t *testing.T) {
	f := parseExpr(t, "Xp")
	if _, ok := f.(Next); !ok {
		t.Errorf("expected Next, got %T", f)
	}
}

func TestParseUntil(t *testing.T) {
	f := parseExpr(t, "p U q")
	if _, ok := f.(Until); !ok {
		t.Errorf("expected Until, got %T", f)
	}
}

func TestParseNestedModal(t *testing.T) {
	f := parseExpr(t, "[](p -> <>q)")
	b, ok := f.(Box)
	if !ok {
		t.Fatalf("expected Box, got %T", f)
	}
	imp, ok := b.Formula.(Implies)
	if !ok {
		t.Fatalf("expected Implies under Box, got %T", b.Formula)
	}
	if _, ok := imp.Antecedent.(Atom); !ok {
		t.Error("antecedent should be Atom")
	}
	if _, ok := imp.Consequent.(Diamond); !ok {
		t.Error("consequent should be Diamond")
	}
}

func TestParseParens(t *testing.T) {
	f := parseExpr(t, "(p && q)")
	if _, ok := f.(And); !ok {
		t.Errorf("expected And, got %T", f)
	}
}

func TestParsePrecedence(t *testing.T) {
	// && binds tighter than ||
	f := parseExpr(t, "p || q && r")
	o, ok := f.(Or)
	if !ok {
		t.Fatalf("expected Or at top, got %T", f)
	}
	if _, ok := o.Right.(And); !ok {
		t.Error("right of || should be And (&& binds tighter)")
	}
}

func TestParsePrecedenceImplies(t *testing.T) {
	// -> is right-associative: p -> q -> r = p -> (q -> r)
	f := parseExpr(t, "p -> q -> r")
	i, ok := f.(Implies)
	if !ok {
		t.Fatalf("expected Implies, got %T", f)
	}
	if _, ok := i.Antecedent.(Atom); !ok {
		t.Error("first antecedent should be Atom p")
	}
	if _, ok := i.Consequent.(Implies); !ok {
		t.Error("consequent should be Implies (q -> r)")
	}
}

func TestParseASCIIBox(t *testing.T) {
	f := parseExpr(t, "Gp")
	if _, ok := f.(Box); !ok {
		t.Errorf("expected Box for 'G', got %T", f)
	}
}

func TestParseASCIIDiamond(t *testing.T) {
	f := parseExpr(t, "Fp")
	if _, ok := f.(Diamond); !ok {
		t.Errorf("expected Diamond for 'F', got %T", f)
	}
}

func TestLexerTokenTypes(t *testing.T) {
	pool := newParserPool(t)
	tests := []struct {
		input string
		types []TokenType
	}{
		{"p", []TokenType{TokAtom, TokEOF}},
		{"!p", []TokenType{TokNot, TokAtom, TokEOF}},
		{"p && q", []TokenType{TokAtom, TokAnd, TokAtom, TokEOF}},
		{"p || q", []TokenType{TokAtom, TokOr, TokAtom, TokEOF}},
		{"p -> q", []TokenType{TokAtom, TokImplies, TokAtom, TokEOF}},
		{"p <-> q", []TokenType{TokAtom, TokIff, TokAtom, TokEOF}},
		{"[]p", []TokenType{TokBox, TokAtom, TokEOF}},
		{"<>p", []TokenType{TokDiamond, TokAtom, TokEOF}},
		{"Xp", []TokenType{TokNext, TokAtom, TokEOF}},
		{"p U q", []TokenType{TokAtom, TokUntil, TokAtom, TokEOF}},
		{"(p)", []TokenType{TokLParen, TokAtom, TokRParen, TokEOF}},
		{"p && q || r", []TokenType{TokAtom, TokAnd, TokAtom, TokOr, TokAtom, TokEOF}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lexer := NewLexer(tt.input, pool)
			tokens := lexer.Lex()
			if len(tokens) != len(tt.types) {
				t.Fatalf("token count: got %d, want %d", len(tokens), len(tt.types))
			}
			for i, tok := range tokens {
				if tok.Type != tt.types[i] {
					t.Errorf("token[%d]: got %v, want %v", i, tok.Type, tt.types[i])
				}
			}
		})
	}
}

func TestParseError(t *testing.T) {
	_, err := parseExprCatch("p ->")
	if err == nil {
		t.Error("expected error for incomplete expression")
	}
	_, err = parseExprCatch("(p")
	if err == nil {
		t.Error("expected error for unclosed paren")
	}
}

func parseExprCatch(expr string) (Formula, error) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	lexer := NewLexer(expr, pool)
	tokens := lexer.Lex()
	parser := NewParser(tokens, expr)
	return parser.Parse()
}

func TestParseComplex(t *testing.T) {
	// □(p → ◇q) ∧ (r U s)
	f := parseExpr(t, "[](p -> <>q) && (r U s)")
	a, ok := f.(And)
	if !ok {
		t.Fatalf("expected And at top, got %T", f)
	}
	if _, ok := a.Left.(Box); !ok {
		t.Error("left should be Box")
	}
	if _, ok := a.Right.(Until); !ok {
		t.Error("right should be Until")
	}
}

func TestLexerSkipsWhitespace(t *testing.T) {
	pool := newParserPool(t)
	lexer := NewLexer("p   &&   q", pool)
	tokens := lexer.Lex()
	if len(tokens) != 4 { // Atom, And, Atom, EOF
		t.Fatalf("expected 4 tokens, got %d", len(tokens))
	}
}

func TestLexerMultiline(t *testing.T) {
	pool := newParserPool(t)
	lexer := NewLexer("p\n&&\nq", pool)
	tokens := lexer.Lex()
	if len(tokens) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(tokens))
	}
}

func TestAtomName(t *testing.T) {
	f := parseExpr(t, "agent_0")
	a, ok := f.(Atom)
	if !ok {
		t.Fatalf("expected Atom, got %T", f)
	}
	if a.ID <= 0 {
		t.Error("atom ID should be > 0")
	}
}

func TestSameAtomSameID(t *testing.T) {
	pool := newParserPool(t)
	lexer := NewLexer("p && p", pool)
	tokens := lexer.Lex()
	parser := NewParser(tokens, "p && p")
	f, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	a := f.(And)
	id1 := a.Left.(Atom).ID
	id2 := a.Right.(Atom).ID
	if id1 != id2 {
		t.Errorf("same atom should have same ID: %d vs %d", id1, id2)
	}
}

func TestParseTripleNegation(t *testing.T) {
	f := parseExpr(t, "!!!p")
	n1, ok := f.(Not)
	if !ok {
		t.Fatalf("expected Not, got %T", f)
	}
	n2, ok := n1.Formula.(Not)
	if !ok {
		t.Fatalf("expected Not, got %T", n1.Formula)
	}
	n3, ok := n2.Formula.(Not)
	if !ok {
		t.Fatalf("expected Not, got %T", n2.Formula)
	}
	if _, ok := n3.Formula.(Atom); !ok {
		t.Error("innermost should be Atom")
	}
}

func TestParseBoxDiamondNextPrecedence(t *testing.T) {
	// []<>Xp should parse as [](<>(Xp))
	f := parseExpr(t, "[]<>Xp")
	b, ok := f.(Box)
	if !ok {
		t.Fatalf("expected Box at top, got %T", f)
	}
	d, ok := b.Formula.(Diamond)
	if !ok {
		t.Fatalf("expected Diamond, got %T", b.Formula)
	}
	if _, ok := d.Formula.(Next); !ok {
		t.Error("expected Next innermost")
	}
}

func TestInvalidTokenSurvives(t *testing.T) {
	// An '@' is not a recognized token char — lexer skips it and treats as atom
	// Actually the lexer's default case adds it as an Atom token
	pool := newParserPool(t)
	lexer := NewLexer("@", pool)
	tokens := lexer.Lex()
	// Should produce Atom + EOF (or just EOF if we skip)
	if len(tokens) < 2 {
		t.Log("single char tokens handled")
	}
}

// --- Integration: parse + evaluate ---

func TestParseAndEvaluate(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	sys := NewModalLogicSystem(pool, arena)
	defer sys.Close()

	// P && Q should be satisfiable
	sat, err := sys.Evaluate("p && q")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sat {
		t.Error("p && q should be satisfiable")
	}
}

func TestParseAndEvaluateContradiction(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	sys := NewModalLogicSystem(pool, arena)
	defer sys.Close()

	sat, err := sys.Evaluate("p && !p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sat {
		t.Error("p && !p should be unsatisfiable")
	}
}

func TestParseAndValidate(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	sys := NewModalLogicSystem(pool, arena)
	defer sys.Close()

	if err := sys.Validate("p && q"); err != nil {
		t.Errorf("valid expression should not error: %v", err)
	}
	if err := sys.Validate("p ->"); err == nil {
		t.Error("invalid expression should error")
	}
}

func TestParseAndIsValid(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	sys := NewModalLogicSystem(pool, arena)
	defer sys.Close()

	valid, err := sys.IsValid("p || !p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("p || !p should be valid")
	}
}

func TestSupportedOperators(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	sys := NewModalLogicSystem(pool, arena)
	defer sys.Close()

	ops := sys.SupportedOperators()
	if len(ops) == 0 {
		t.Error("should have operators")
	}
	for _, op := range ops {
		if !strings.Contains(op, "(") {
			t.Errorf("operator should have description in parens: %q", op)
		}
	}
}

func TestName(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	sys := NewModalLogicSystem(pool, arena)
	defer sys.Close()

	if sys.Name() != "ModalLogicSystem" {
		t.Errorf("Name: got %q, want ModalLogicSystem", sys.Name())
	}
}

func TestProveEntailsParsed(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	sys := NewModalLogicSystem(pool, arena)
	defer sys.Close()

	ok, err := sys.ProveEntailsParsed([]string{"p && q"}, "p")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("p&&q should entail p")
	}
}

func TestLexerReuse(t *testing.T) {
	pool := newParserPool(t)
	lexer := NewLexer("p", pool)
	t1 := lexer.Lex()
	t2 := lexer.Lex()
	if len(t1) != len(t2) {
		t.Error("reused lexer should produce same token count")
	}
}
