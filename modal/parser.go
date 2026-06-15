package modal

import (
	"fmt"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

// TokenType identifies a lexical token.
type TokenType uint8

const (
	TokEOF TokenType = iota
	TokAtom
	TokNot
	TokAnd
	TokOr
	TokImplies
	TokIff
	TokBox
	TokDiamond
	TokNext
	TokUntil
	TokLParen
	TokRParen
	TokComma
)

// Token is a zero-allocation lexical token — byte offsets into the source string.
type Token struct {
	Type  TokenType
	Start int
	End   int
}

// Lexer performs zero-allocation tokenization of a modal formula string.
// Tokens are stored in a Pool-backed slice. The source string is never copied.
type Lexer struct {
	src    string
	pos    int
	tokens []Token // Pool-backed
	pool   *memory.Pool
}

// NewLexer creates a Lexer backed by the given pool.
func NewLexer(src string, pool *memory.Pool) *Lexer {
	tokens := memory.MustPoolSlice[Token](pool, 64)
	tokens = tokens[:0]
	return &Lexer{src: src, tokens: tokens, pool: pool}
}

// Lex tokenizes the source and returns the token slice.
func (l *Lexer) Lex() []Token {
	l.pos = 0
	l.tokens = l.tokens[:0]
	for l.pos < len(l.src) {
		l.skipSpace()
		if l.pos >= len(l.src) {
			break
		}
		l.tokens = append(l.tokens, l.nextToken())
	}
	l.tokens = append(l.tokens, Token{Type: TokEOF, Start: l.pos, End: l.pos})
	return l.tokens
}

// skipSpace advances past whitespace.
func (l *Lexer) skipSpace() {
	for l.pos < len(l.src) && (l.src[l.pos] == ' ' || l.src[l.pos] == '\t' || l.src[l.pos] == '\n') {
		l.pos++
	}
}

// nextToken reads the next token from position l.pos.
func (l *Lexer) nextToken() Token {
	ch := l.src[l.pos]

	// Single-char tokens
	switch ch {
	case '(':
		l.pos++
		return Token{TokLParen, l.pos - 1, l.pos}
	case ')':
		l.pos++
		return Token{TokRParen, l.pos - 1, l.pos}
	case ',':
		l.pos++
		return Token{TokComma, l.pos - 1, l.pos}
	case '!':
		l.pos++
		return Token{TokNot, l.pos - 1, l.pos}
	}

	// Multi-char tokens
	if ch == '[' && l.pos+1 < len(l.src) && l.src[l.pos+1] == ']' {
		l.pos += 2
		return Token{TokBox, l.pos - 2, l.pos}
	}
	if ch == '<' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '>' {
		l.pos += 2
		return Token{TokDiamond, l.pos - 2, l.pos}
	}
	if ch == '-' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '>' {
		l.pos += 2
		return Token{TokImplies, l.pos - 2, l.pos}
	}
	if ch == '<' && l.pos+2 < len(l.src) && l.src[l.pos+1] == '-' && l.src[l.pos+2] == '>' {
		l.pos += 3
		return Token{TokIff, l.pos - 3, l.pos}
	}
	if ch == '&' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '&' {
		l.pos += 2
		return Token{TokAnd, l.pos - 2, l.pos}
	}
	if ch == '|' && l.pos+1 < len(l.src) && l.src[l.pos+1] == '|' {
		l.pos += 2
		return Token{TokOr, l.pos - 2, l.pos}
	}

	// Single-char ASCII operators
	if ch == 'X' || ch == 'x' {
		l.pos++
		return Token{TokNext, l.pos - 1, l.pos}
	}
	if ch == 'U' || ch == 'u' {
		l.pos++
		return Token{TokUntil, l.pos - 1, l.pos}
	}

	// ASCII modalities
	if ch == 'G' || ch == 'g' {
		l.pos++
		return Token{TokBox, l.pos - 1, l.pos}
	}
	if ch == 'F' || ch == 'f' {
		l.pos++
		return Token{TokDiamond, l.pos - 1, l.pos}
	}

	// Identifiers (atoms, agent names)
	if isIdentStart(ch) {
		start := l.pos
		l.pos++
		for l.pos < len(l.src) && isIdentPart(l.src[l.pos]) {
			l.pos++
		}
		return Token{TokAtom, start, l.pos}
	}

	// Skip unknown char
	l.pos++
	return Token{TokAtom, l.pos - 1, l.pos}
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentPart(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}

// Parser is a Pratt parser for modal logic formulas.
// Uses an explicit Pool-backed token slice with a position cursor.
// When a Registry is set, parsed formulas are hash-consed for O(1) equality.
type Parser struct {
	tokens []Token
	pos    int
	src    string
	sym    *symbolTable
	reg    *Registry // optional hash cons registry
}

// SetRegistry attaches a formula hash cons registry. When set, all parsed
// formulas are canonicalized through the registry before being returned.
func (p *Parser) SetRegistry(reg *Registry) { p.reg = reg }

// symbolTable maps atom names to formula IDs during parsing.
type symbolTable struct {
	names map[string]fuzzy.VarID
	next  fuzzy.VarID
}

func newSymbolTable() *symbolTable {
	return &symbolTable{
		names: make(map[string]fuzzy.VarID),
		next:  1,
	}
}

func (s *symbolTable) intern(name string) fuzzy.VarID {
	if id, ok := s.names[name]; ok {
		return id
	}
	id := s.next
	s.next++
	s.names[name] = id
	return id
}

// NewParser creates a Parser for the given tokens and source.
func NewParser(tokens []Token, src string) *Parser {
	return &Parser{
		tokens: tokens,
		src:    src,
		sym:    newSymbolTable(),
	}
}

// Parse parses the token stream into a Formula.
// CC=7.
func (p *Parser) Parse() (Formula, error) {
	f, err := p.parseExpr(0)
	if err != nil {
		return nil, err
	}
	if p.peek().Type != TokEOF {
		return nil, fmt.Errorf("unexpected token after expression")
	}
	return p.canon(f), nil
}

// canon returns the canonical form of f via the registry, or f if no registry is set.
func (p *Parser) canon(f Formula) Formula {
	if p.reg != nil {
		return p.reg.Intern(f)
	}
	return f
}

// parseExpr is the Pratt parsing core. minBP is the minimum binding power
// required to continue parsing (left-associative operators use minBP,
// right-associative use minBP-1). CC=5.
func (p *Parser) parseExpr(minBP int) (Formula, error) {
	tok := p.next()
	left, err := p.parsePrefix(tok)
	if err != nil {
		return nil, err
	}

	for {
		peek := p.peek()
		bp := infixBP(peek.Type)
		if bp == 0 || bp <= minBP {
			break
		}
		tok = p.next()
		left, err = p.parseInfix(tok, left, bp)
		if err != nil {
			return nil, err
		}
	}
	return left, nil
}

// parsePrefix handles prefix (nud) expressions. CC=6.
func (p *Parser) parsePrefix(tok Token) (Formula, error) {
	switch tok.Type {
	case TokAtom:
		name := p.src[tok.Start:tok.End]
		return p.canon(Atom{ID: p.sym.intern(name)}), nil

	case TokNot:
		operand, err := p.parseExpr(60)
		if err != nil {
			return nil, err
		}
		return p.canon(Not{Formula: operand}), nil

	case TokBox:
		operand, err := p.parseExpr(70)
		if err != nil {
			return nil, err
		}
		return p.canon(Box{Formula: operand, Rel: RelCausal}), nil

	case TokDiamond:
		operand, err := p.parseExpr(70)
		if err != nil {
			return nil, err
		}
		return p.canon(Diamond{Formula: operand, Rel: RelCausal}), nil

	case TokNext:
		operand, err := p.parseExpr(70)
		if err != nil {
			return nil, err
		}
		return p.canon(Next{Formula: operand}), nil

	case TokLParen:
		inner, err := p.parseExpr(0)
		if err != nil {
			return nil, err
		}
		if p.peek().Type != TokRParen {
			return nil, fmt.Errorf("expected ')'")
		}
		p.next()
		return inner, nil

	default:
		return nil, fmt.Errorf("unexpected token: %v", tok.Type)
	}
}

// parseInfix handles infix (led) operators. CC=4.
func (p *Parser) parseInfix(tok Token, left Formula, bp int) (Formula, error) {
	switch tok.Type {
	case TokAnd:
		right, err := p.parseExpr(bp)
		if err != nil {
			return nil, err
		}
		return p.canon(And{Left: left, Right: right}), nil

	case TokOr:
		right, err := p.parseExpr(bp)
		if err != nil {
			return nil, err
		}
		return p.canon(Or{Left: left, Right: right}), nil

	case TokImplies:
		right, err := p.parseExpr(bp - 1)
		if err != nil {
			return nil, err
		}
		return p.canon(Implies{Antecedent: left, Consequent: right}), nil

	case TokIff:
		right, err := p.parseExpr(bp)
		if err != nil {
			return nil, err
		}
		return p.canon(Iff{Left: left, Right: right}), nil

	case TokUntil:
		right, err := p.parseExpr(bp)
		if err != nil {
			return nil, err
		}
		return p.canon(Until{Left: left, Right: right}), nil

	default:
		return nil, fmt.Errorf("unexpected infix token: %v", tok.Type)
	}
}

// Next represents the temporal Next operator ○.
type Next struct{ Formula Formula }

func (n Next) Evaluate(w World, m *Model) (TruthValue, error) {
	return 0.0, fmt.Errorf("Next requires TemporalModel, not plain Model")
}

// Until represents the temporal Until operator U.
type Until struct{ Left, Right Formula }

func (u Until) Evaluate(w World, m *Model) (TruthValue, error) {
	return 0.0, fmt.Errorf("Until requires TemporalModel, not plain Model")
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) next() Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

// infixBP returns the left binding power for infix tokens.
// Higher numbers bind tighter.
func infixBP(t TokenType) int {
	switch t {
	case TokIff:
		return 10
	case TokImplies:
		return 20
	case TokUntil:
		return 25
	case TokOr:
		return 30
	case TokAnd:
		return 40
	}
	return 0
}
