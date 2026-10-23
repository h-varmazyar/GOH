package parser

type TokenType int

const (
	TokenIdentifier TokenType = iota
	TokenNumber
	TokenBool
	TokenOperator
	TokenDot
	TokenLParen
	TokenRParen
	TokenComma
	TokenEOF
)

type Token struct {
	Type  TokenType
	Value string
}
