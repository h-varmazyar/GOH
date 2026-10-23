package parser

import (
	"fmt"

	"github.com/h-varmazyar/gatec/pkg/strategy/ast"
	"github.com/shopspring/decimal"
)

type Parser struct {
	tokens  []Token
	current int
}

func New(input string) (*Parser, error) {
	tokens, err := Lex(input)
	if err != nil {
		return nil, err
	}
	return &Parser{tokens: tokens}, nil
}

func (p *Parser) ParseExpression() (ast.Expression, error) {
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if !p.isAtEnd() {
		return nil, fmt.Errorf("unexpected token %q", p.peek().Value)
	}
	return expr, nil
}

func (p *Parser) parseOr() (ast.Expression, error)  { return p.parseBinary(p.parseAnd, "||") }
func (p *Parser) parseAnd() (ast.Expression, error) { return p.parseBinary(p.parseEquality, "&&") }
func (p *Parser) parseEquality() (ast.Expression, error) {
	return p.parseBinary(p.parseComparison, "==", "!=")
}
func (p *Parser) parseComparison() (ast.Expression, error) {
	return p.parseBinary(p.parseTerm, "<", "<=", ">", ">=")
}
func (p *Parser) parseTerm() (ast.Expression, error)   { return p.parseBinary(p.parseFactor, "+", "-") }
func (p *Parser) parseFactor() (ast.Expression, error) { return p.parseBinary(p.parseUnary, "*", "/") }

func (p *Parser) parseUnary() (ast.Expression, error) {
	if p.matchOperator("!", "-") {
		op := p.previous().Value
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &ast.UnaryExpression{Operator: op, Expr: right}, nil
	}
	return p.parsePrimary()
}

func (p *Parser) parsePrimary() (ast.Expression, error) {
	if p.match(TokenBool) {
		return &ast.BoolLiteral{Value: p.previous().Value == "true"}, nil
	}
	if p.match(TokenNumber) {
		d, err := decimal.NewFromString(p.previous().Value)
		if err != nil {
			return nil, err
		}
		return &ast.Literal{Value: d}, nil
	}
	if p.match(TokenIdentifier) {
		name := p.previous().Value
		if p.match(TokenLParen) {
			return p.parseFunctionCall(name)
		}
		path := []string{name}
		for p.match(TokenDot) {
			if !p.match(TokenIdentifier) {
				return nil, fmt.Errorf("expected identifier after dot")
			}
			path = append(path, p.previous().Value)
		}
		return &ast.Identifier{Path: path}, nil
	}
	if p.match(TokenLParen) {
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if !p.match(TokenRParen) {
			return nil, fmt.Errorf("expected ')' after expression")
		}
		return expr, nil
	}
	return nil, fmt.Errorf("unexpected token %q", p.peek().Value)
}

func (p *Parser) parseFunctionCall(name string) (ast.Expression, error) {
	args := make([]ast.Expression, 0)
	if !p.check(TokenRParen) {
		for {
			arg, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
			if !p.match(TokenComma) {
				break
			}
		}
	}
	if !p.match(TokenRParen) {
		return nil, fmt.Errorf("expected ')' after function args")
	}
	return &ast.FunctionCall{Name: name, Args: args}, nil
}

func (p *Parser) parseBinary(next func() (ast.Expression, error), operators ...string) (ast.Expression, error) {
	left, err := next()
	if err != nil {
		return nil, err
	}
	for p.matchOperator(operators...) {
		op := p.previous().Value
		right, err := next()
		if err != nil {
			return nil, err
		}
		left = &ast.BinaryExpression{Left: left, Operator: op, Right: right}
	}
	return left, nil
}

func (p *Parser) match(tt TokenType) bool {
	if p.check(tt) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) matchOperator(values ...string) bool {
	if p.peek().Type != TokenOperator {
		return false
	}
	for _, value := range values {
		if p.peek().Value == value {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) check(tt TokenType) bool {
	if p.isAtEnd() {
		return tt == TokenEOF
	}
	return p.peek().Type == tt
}

func (p *Parser) advance() Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.previous()
}

func (p *Parser) isAtEnd() bool   { return p.peek().Type == TokenEOF }
func (p *Parser) peek() Token     { return p.tokens[p.current] }
func (p *Parser) previous() Token { return p.tokens[p.current-1] }
