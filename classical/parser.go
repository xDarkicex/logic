package classical

import (
	"fmt"

	"github.com/xDarkicex/logic/core"
)

// Parser implements recursive descent parsing for logical expressions
type Parser struct {
	tokens  []Token
	current int
}

// ParseExpression parses a logical expression string into an AST
func ParseExpression(expr string) (*ASTNode, error) {
	lexer := NewLexer(expr)
	tokens := lexer.Lex()

	// Check for lexical errors
	for _, token := range tokens {
		if token.Type == TokenError {
			return nil, core.NewLogicError("classical", "ParseExpression",
				fmt.Sprintf("invalid character '%s' at position %d", token.Value, token.Position))
		}
	}

	parser := &Parser{tokens: tokens, current: 0}
	ast, err := parser.parseExpression()
	if err != nil {
		return nil, err
	}

	// Check for trailing tokens
	if !parser.isAtEnd() {
		return nil, core.NewLogicError("classical", "ParseExpression",
			fmt.Sprintf("unexpected token '%s' at position %d",
				parser.peek().Value, parser.peek().Position))
	}

	return ast, nil
}

// parseExpression parses the highest precedence level (implication/iff)
func (p *Parser) parseExpression() (*ASTNode, error) {
	return p.parseIff()
}

// parseIff parses biconditional (lowest precedence)
func (p *Parser) parseIff() (*ASTNode, error) {
	expr, err := p.parseImplication()
	if err != nil {
		return nil, err
	}

	for p.match(TokenIff) {
		right, err := p.parseImplication()
		if err != nil {
			return nil, err
		}
		expr = &ASTNode{
			Type:     NodeIff,
			Children: []*ASTNode{expr, right},
			Position: p.previous().Position,
		}
	}

	return expr, nil
}

// parseImplication parses implication (right associative)
func (p *Parser) parseImplication() (*ASTNode, error) {
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	if p.match(TokenImplies) {
		right, err := p.parseImplication() // Right associative
		if err != nil {
			return nil, err
		}
		expr = &ASTNode{
			Type:     NodeImplies,
			Children: []*ASTNode{expr, right},
			Position: p.previous().Position,
		}
	}

	return expr, nil
}

// parseOr parses OR operations
func (p *Parser) parseOr() (*ASTNode, error) {
	expr, err := p.parseXor()
	if err != nil {
		return nil, err
	}

	for p.match(TokenOr, TokenNor) {
		operator := p.previous()
		right, err := p.parseXor()
		if err != nil {
			return nil, err
		}

		nodeType := NodeOr
		if operator.Type == TokenNor {
			nodeType = NodeNor
		}

		expr = &ASTNode{
			Type:     nodeType,
			Children: []*ASTNode{expr, right},
			Position: operator.Position,
		}
	}

	return expr, nil
}

// parseXor parses XOR operations
func (p *Parser) parseXor() (*ASTNode, error) {
	expr, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.match(TokenXor) {
		operator := p.previous()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}

		expr = &ASTNode{
			Type:     NodeXor,
			Children: []*ASTNode{expr, right},
			Position: operator.Position,
		}
	}

	return expr, nil
}

// parseAnd parses AND operations
func (p *Parser) parseAnd() (*ASTNode, error) {
	expr, err := p.parseUnary()
	if err != nil {
		return nil, err
	}

	for p.match(TokenAnd, TokenNand) {
		operator := p.previous()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}

		nodeType := NodeAnd
		if operator.Type == TokenNand {
			nodeType = NodeNand
		}

		expr = &ASTNode{
			Type:     nodeType,
			Children: []*ASTNode{expr, right},
			Position: operator.Position,
		}
	}

	return expr, nil
}

// parseUnary parses unary operations (NOT)
func (p *Parser) parseUnary() (*ASTNode, error) {
	if p.match(TokenNot) {
		operator := p.previous()
		expr, err := p.parseUnary()
		if err != nil {
			return nil, err
		}

		return &ASTNode{
			Type:     NodeNot,
			Children: []*ASTNode{expr},
			Position: operator.Position,
		}, nil
	}

	return p.parsePrimary()
}

// parsePrimary parses primary expressions (variables, constants, parentheses)
func (p *Parser) parsePrimary() (*ASTNode, error) {
	if p.match(TokenConstant) {
		token := p.previous()
		return &ASTNode{
			Type:     NodeConstant,
			Value:    token.Value,
			Position: token.Position,
		}, nil
	}

	if p.match(TokenVariable) {
		token := p.previous()
		return &ASTNode{
			Type:     NodeVariable,
			Value:    token.Value,
			Position: token.Position,
		}, nil
	}

	if p.match(TokenLeftParen) {
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		if !p.match(TokenRightParen) {
			return nil, core.NewLogicError("classical", "Parser.parsePrimary",
				fmt.Sprintf("expected ')' at position %d", p.peek().Position))
		}

		return expr, nil
	}

	return nil, core.NewLogicError("classical", "Parser.parsePrimary",
		fmt.Sprintf("expected expression at position %d", p.peek().Position))
}

// Helper methods for parser state management
func (p *Parser) match(types ...TokenType) bool {
	for _, tokenType := range types {
		if p.check(tokenType) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) check(tokenType TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Type == tokenType
}

func (p *Parser) advance() Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.previous()
}

func (p *Parser) isAtEnd() bool {
	return p.peek().Type == TokenEOF
}

func (p *Parser) peek() Token {
	return p.tokens[p.current]
}

func (p *Parser) previous() Token {
	return p.tokens[p.current-1]
}
