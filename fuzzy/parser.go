package fuzzy

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xDarkicex/memory"
)

// TokenType represents a lexer token type for fuzzy syntax.
type TokenType int

const (
	TokenIf TokenType = iota
	TokenThen
	TokenIs
	TokenAnd
	TokenNot
	TokenWeight
	TokenLeftParen
	TokenRightParen
	TokenIdentifier
	TokenNumber
	TokenEOF
	TokenError
)

// Token represents a single lexical token.
type Token struct {
	Type     TokenType
	Value    string
	Position int
}

// Lexer tokenizes fuzzy rule strings.
type Lexer struct {
	input    string
	position int
	tokens   []Token
	pool     *memory.Pool
}

// NewLexer creates a new Lexer. pool must be non-nil.
func NewLexer(input string, pool *memory.Pool) *Lexer {
	return &Lexer{
		input:  input,
		pool:   pool,
		tokens: memory.MustPoolSlice[Token](pool, 0),
	}
}

func (l *Lexer) appendToken(t Token) {
	if len(l.tokens) == cap(l.tokens) {
		newCap := cap(l.tokens) * 2
		if newCap == 0 {
			newCap = 16
		}
		newSlice := memory.MustPoolSlice[Token](l.pool, newCap)
		l.tokens = append(newSlice, l.tokens...)
	}
	l.tokens = append(l.tokens, t)
}

// Lex performs the tokenization.
func (l *Lexer) Lex() []Token {
	for l.position < len(l.input) {
		l.skipWhitespace()
		if l.position >= len(l.input) {
			break
		}

		tok := l.nextToken()
		l.appendToken(tok)

		if tok.Type == TokenError {
			break
		}
	}
	l.appendToken(Token{Type: TokenEOF, Position: l.position})
	return l.tokens
}

func (l *Lexer) skipWhitespace() {
	for l.position < len(l.input) && unicode.IsSpace(rune(l.input[l.position])) {
		l.position++
	}
}

func (l *Lexer) nextToken() Token {
	start := l.position
	r, size := utf8.DecodeRuneInString(l.input[l.position:])
	
	if r == '(' {
		l.position += size
		return Token{Type: TokenLeftParen, Value: "(", Position: start}
	}
	if r == ')' {
		l.position += size
		return Token{Type: TokenRightParen, Value: ")", Position: start}
	}

	if unicode.IsDigit(r) || r == '.' || r == '-' {
		return l.readNumber(start)
	}

	if unicode.IsLetter(r) || r == '_' {
		return l.readIdentifier(start)
	}

	return Token{Type: TokenError, Value: string(r), Position: start}
}

func (l *Lexer) readNumber(start int) Token {
	hasDot := false
	for l.position < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.position:])
		if unicode.IsDigit(r) || (r == '.' && !hasDot) || (r == '-' && l.position == start) {
			if r == '.' {
				hasDot = true
			}
			l.position += size
		} else {
			break
		}
	}
	val := l.input[start:l.position]
	if val == "-" || val == "." {
		return Token{Type: TokenError, Value: val, Position: start}
	}
	return Token{Type: TokenNumber, Value: val, Position: start}
}

func (l *Lexer) readIdentifier(start int) Token {
	for l.position < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.position:])
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			l.position += size
		} else {
			break
		}
	}

	val := l.input[start:l.position]
	lower := strings.ToLower(val)
	tokType := TokenIdentifier

	switch lower {
	case "if":
		tokType = TokenIf
	case "then":
		tokType = TokenThen
	case "is":
		tokType = TokenIs
	case "and":
		tokType = TokenAnd
	case "not":
		tokType = TokenNot
	case "weight":
		tokType = TokenWeight
	}

	return Token{Type: tokType, Value: val, Position: start}
}

// Parser parses fuzzy rules.
type Parser struct {
	tokens  []Token
	pos     int
	symbols *SymbolTable
	arena   *memory.Arena
}

// NewParser creates a new fuzzy parser.
func NewParser(tokens []Token, symbols *SymbolTable, arena *memory.Arena) *Parser {
	return &Parser{
		tokens:  tokens,
		symbols: symbols,
		arena:   arena,
	}
}

// ParseCondition parses a fuzzy condition string (e.g. "Var IS Term AND Var IS NOT Term")
// Time: O(t), CC: 8
func (p *Parser) ParseCondition() ([]FuzzyCondition, error) {
	var conds []FuzzyCondition
	for {
		vTok, err := p.expect(TokenIdentifier)
		if err != nil {
			return nil, err
		}
		_, err = p.expect(TokenIs)
		if err != nil {
			return nil, err
		}

		negated := false
		if p.current().Type == TokenNot {
			negated = true
			p.advance()
		}

		tTok, err := p.expect(TokenIdentifier)
		if err != nil {
			return nil, err
		}

		varID := p.symbols.Register(vTok.Value, p.arena)
		termID := p.symbols.Register(tTok.Value, p.arena)
		
		conds = append(conds, FuzzyCondition{Variable: varID, Term: termID, Negated: negated})

		if p.current().Type == TokenAnd {
			p.advance()
			continue
		} else if p.current().Type == TokenEOF {
			break
		} else {
			return nil, fmt.Errorf("expected AND or EOF at position %d, got %v", p.current().Position, p.current().Type)
		}
	}
	return conds, nil
}

// ParseRule parses a rule formatted like:
// IF var IS [NOT] term AND var IS term THEN outVar IS outTerm [(WEIGHT 0.8)]
// Time: O(t), CC: 8
func (p *Parser) ParseRule() (*FuzzyRule, error) {
	if p.current().Type != TokenIf {
		return nil, fmt.Errorf("expected IF at position %d", p.current().Position)
	}
	p.advance()

	rule := NewFuzzyRule(1.0)

	// Parse antecedents
	for {
		vTok, err := p.expect(TokenIdentifier)
		if err != nil {
			return nil, err
		}
		_, err = p.expect(TokenIs)
		if err != nil {
			return nil, err
		}

		negated := false
		if p.current().Type == TokenNot {
			negated = true
			p.advance()
		}

		tTok, err := p.expect(TokenIdentifier)
		if err != nil {
			return nil, err
		}

		varID := p.symbols.Register(vTok.Value, p.arena)
		termID := p.symbols.Register(tTok.Value, p.arena)
		rule.AddAntecedent(varID, termID, negated)

		if p.current().Type == TokenAnd {
			p.advance()
			continue
		} else if p.current().Type == TokenThen {
			break
		} else {
			return nil, fmt.Errorf("expected AND or THEN at position %d", p.current().Position)
		}
	}

	p.advance() // Consume THEN

	// Parse consequent
	outVTok, err := p.expect(TokenIdentifier)
	if err != nil {
		return nil, err
	}
	_, err = p.expect(TokenIs)
	if err != nil {
		return nil, err
	}
	outTTok, err := p.expect(TokenIdentifier)
	if err != nil {
		return nil, err
	}

	outVarID := p.symbols.Register(outVTok.Value, p.arena)
	outTermID := p.symbols.Register(outTTok.Value, p.arena)
	rule.SetConsequent(outVarID, outTermID)

	// Optional weight
	if p.current().Type == TokenLeftParen {
		p.advance()
		if p.current().Type == TokenWeight {
			p.advance()
			wTok, err := p.expect(TokenNumber)
			if err != nil {
				return nil, err
			}
			weight, err := strconv.ParseFloat(wTok.Value, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid weight %s", wTok.Value)
			}
			rule.Weight = TruthValue(weight)
		}
		_, err = p.expect(TokenRightParen)
		if err != nil {
			return nil, err
		}
	}

	return rule, nil
}

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1] // EOF
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

func (p *Parser) expect(t TokenType) (Token, error) {
	if p.current().Type != t {
		return Token{}, fmt.Errorf("expected token type %v, got %v at position %d", t, p.current().Type, p.current().Position)
	}
	tok := p.current()
	p.advance()
	return tok, nil
}
