package classical

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenType represents different types of tokens in logical expressions
type TokenType int

const (
	TokenVariable TokenType = iota
	TokenConstant
	TokenAnd
	TokenOr
	TokenXor
	TokenNot
	TokenNand
	TokenNor
	TokenImplies
	TokenIff
	TokenLeftParen
	TokenRightParen
	TokenEOF
	TokenError
)

// String returns string representation of TokenType
func (tt TokenType) String() string {
	switch tt {
	case TokenVariable:
		return "Variable"
	case TokenConstant:
		return "Constant"
	case TokenAnd:
		return "And"
	case TokenOr:
		return "Or"
	case TokenXor:
		return "Xor"
	case TokenNot:
		return "Not"
	case TokenNand:
		return "Nand"
	case TokenNor:
		return "Nor"
	case TokenImplies:
		return "Implies"
	case TokenIff:
		return "Iff"
	case TokenLeftParen:
		return "LeftParen"
	case TokenRightParen:
		return "RightParen"
	case TokenEOF:
		return "EOF"
	case TokenError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Token represents a single token in a logical expression
type Token struct {
	Type     TokenType
	Value    string
	Position int
}

// Lexer tokenizes logical expressions
type Lexer struct {
	input    string
	position int
	tokens   []Token
}

// NewLexer creates a new lexer for the given input
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		tokens: make([]Token, 0),
	}
}

// Lex tokenizes the input and returns all tokens
func (l *Lexer) Lex() []Token {
	for l.position < len(l.input) {
		l.skipWhitespace()
		if l.position >= len(l.input) {
			break
		}

		token := l.nextToken()
		l.tokens = append(l.tokens, token)

		if token.Type == TokenError {
			break
		}
	}

	l.tokens = append(l.tokens, Token{Type: TokenEOF, Position: l.position})
	return l.tokens
}

// skipWhitespace skips whitespace characters
func (l *Lexer) skipWhitespace() {
	for l.position < len(l.input) && unicode.IsSpace(rune(l.input[l.position])) {
		l.position++
	}
}

// nextToken reads and returns the next token
func (l *Lexer) nextToken() Token {
	if l.position >= len(l.input) {
		return Token{Type: TokenEOF, Position: l.position}
	}

	start := l.position

	// Handle UTF-8 characters properly
	r, size := utf8.DecodeRuneInString(l.input[l.position:])
	if size == 0 {
		return Token{Type: TokenError, Value: "invalid UTF-8", Position: start}
	}

	switch r {
	case '(':
		l.position++
		return Token{Type: TokenLeftParen, Value: "(", Position: start}
	case ')':
		l.position++
		return Token{Type: TokenRightParen, Value: ")", Position: start}
	case '&':
		l.position++
		return Token{Type: TokenAnd, Value: "&", Position: start}
	case '|':
		l.position++
		return Token{Type: TokenOr, Value: "|", Position: start}
	case '^':
		l.position++
		return Token{Type: TokenXor, Value: "^", Position: start}
	case '!':
		l.position++
		return Token{Type: TokenNot, Value: "!", Position: start}
	case '¬':
		l.position += size
		return Token{Type: TokenNot, Value: "¬", Position: start}
	case '∧':
		l.position += size
		return Token{Type: TokenAnd, Value: "∧", Position: start}
	case '∨':
		l.position += size
		return Token{Type: TokenOr, Value: "∨", Position: start}
	case '⊕':
		l.position += size
		return Token{Type: TokenXor, Value: "⊕", Position: start}
	case '→':
		l.position += size
		return Token{Type: TokenImplies, Value: "→", Position: start}
	case '↔':
		l.position += size
		return Token{Type: TokenIff, Value: "↔", Position: start}
	case '-':
		if l.position+1 < len(l.input) && l.input[l.position+1] == '>' {
			l.position += 2
			return Token{Type: TokenImplies, Value: "->", Position: start}
		}
		return Token{Type: TokenError, Value: string(r), Position: start}
	case '<':
		if l.position+2 < len(l.input) && l.input[l.position+1:l.position+3] == "->" {
			l.position += 3
			return Token{Type: TokenIff, Value: "<->", Position: start}
		}
		return Token{Type: TokenError, Value: string(r), Position: start}
	default:
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return l.readIdentifier(start)
		}
		return Token{Type: TokenError, Value: string(r), Position: start}
	}
}

// readIdentifier reads an identifier (variable name or keyword)
func (l *Lexer) readIdentifier(start int) Token {
	for l.position < len(l.input) {
		r, size := utf8.DecodeRuneInString(l.input[l.position:])
		if size == 0 {
			break
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			l.position += size
		} else {
			break
		}
	}

	value := l.input[start:l.position]
	tokenType := l.identifierType(value)

	return Token{Type: tokenType, Value: value, Position: start}
}

// identifierType determines the token type for an identifier
func (l *Lexer) identifierType(value string) TokenType {
	lower := strings.ToLower(value)

	switch lower {
	case "and":
		return TokenAnd
	case "or":
		return TokenOr
	case "xor":
		return TokenXor
	case "not":
		return TokenNot
	case "nand":
		return TokenNand
	case "nor":
		return TokenNor
	case "implies":
		return TokenImplies
	case "iff":
		return TokenIff
	case "true", "t", "1":
		return TokenConstant
	case "false", "f", "0":
		return TokenConstant
	default:
		return TokenVariable
	}
}
