package fuzzy

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func TestLexer(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Free()

	input := "IF Temp IS Hot AND Pressure IS NOT Low THEN Valve IS Open (WEIGHT 0.85)"
	lexer := NewLexer(input, pool)
	tokens := lexer.Lex()

	expectedTokens := []TokenType{
		TokenIf, TokenIdentifier, TokenIs, TokenIdentifier,
		TokenAnd, TokenIdentifier, TokenIs, TokenNot, TokenIdentifier,
		TokenThen, TokenIdentifier, TokenIs, TokenIdentifier,
		TokenLeftParen, TokenWeight, TokenNumber, TokenRightParen,
		TokenEOF,
	}

	if len(tokens) != len(expectedTokens) {
		t.Fatalf("Expected %d tokens, got %d", len(expectedTokens), len(tokens))
	}

	for i, expected := range expectedTokens {
		if tokens[i].Type != expected {
			t.Errorf("Token %d: expected %v, got %v (%q)", i, expected, tokens[i].Type, tokens[i].Value)
		}
	}

	// Lexer edge cases
	l2 := NewLexer("+-()", nil) // no pool coverage
	toks2 := l2.Lex()
	if toks2[0].Type != TokenError {
		t.Error("Expected error for standalone '+'")
	}
	
	l3 := NewLexer(".-5", nil)
	toks3 := l3.Lex()
	if toks3[0].Type != TokenError {
		t.Error("Expected error for invalid float '.'")
	}

	// Token test
	l4 := NewLexer(" \n\t ", nil)
	toks4 := l4.Lex()
	if toks4[0].Type != TokenEOF {
		t.Error("Expected EOF after whitespace")
	}
}

func TestParser(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Free()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()
	sym := NewSymbolTable(arena)

	input := "IF Temp IS Hot AND Pressure IS NOT Low THEN Valve IS Open (WEIGHT 0.85)"
	lexer := NewLexer(input, pool)
	tokens := lexer.Lex()

	parser := NewParser(tokens, sym, arena)
	rule, err := parser.ParseRule()
	if err != nil {
		t.Fatalf("Failed to parse rule: %v", err)
	}

	if rule == nil {
		t.Fatal("Expected rule, got nil")
	}

	if len(rule.Antecedents) != 2 {
		t.Errorf("Expected 2 antecedents, got %d", len(rule.Antecedents))
	}

	if rule.Antecedents[1].Negated != true {
		t.Errorf("Expected second antecedent to be negated")
	}

	if rule.Weight != 0.85 {
		t.Errorf("Expected weight 0.85, got %v", rule.Weight)
	}
}

func TestParserErrors(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Free()
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()
	sym := NewSymbolTable(arena)

	tests := []struct {
		name  string
		input string
	}{
		{"Missing IF", "Temp IS Hot THEN Valve IS Open"},
		{"Missing IS", "IF Temp Hot THEN Valve IS Open"},
		{"Missing THEN", "IF Temp IS Hot Valve IS Open"},
		{"Missing Consequent IS", "IF Temp IS Hot THEN Valve Open"},
		{"Bad Weight", "IF Temp IS Hot THEN Valve IS Open (WEIGHT x)"},
		{"Bad Paren", "IF Temp IS Hot THEN Valve IS Open (WEIGHT 0.5"},
		{"Missing And/Then", "IF Temp IS Hot Pressure IS Low THEN Valve IS Open"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lexer := NewLexer(tc.input, pool)
			tokens := lexer.Lex()
			parser := NewParser(tokens, sym, arena)
			_, err := parser.ParseRule()
			if err == nil {
				t.Errorf("Expected error for input: %q", tc.input)
			}
		})
	}
}
