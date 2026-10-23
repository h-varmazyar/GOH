package parser

import (
	"fmt"
	"strings"
	"unicode"
)

func Lex(input string) ([]Token, error) {
	var tokens []Token
	runes := []rune(input)

	for i := 0; i < len(runes); {
		r := runes[i]
		if unicode.IsSpace(r) {
			i++
			continue
		}

		if isIdentifierStart(r) {
			start := i
			i++
			for i < len(runes) && isIdentifierPart(runes[i]) {
				i++
			}
			value := string(runes[start:i])
			switch strings.ToLower(value) {
			case "true", "false":
				tokens = append(tokens, Token{Type: TokenBool, Value: strings.ToLower(value)})
			default:
				tokens = append(tokens, Token{Type: TokenIdentifier, Value: value})
			}
			continue
		}

		if unicode.IsDigit(r) {
			start := i
			i++
			dotCount := 0
			for i < len(runes) {
				if runes[i] == '.' {
					dotCount++
					if dotCount > 1 {
						return nil, fmt.Errorf("invalid number at position %d", i)
					}
					i++
					continue
				}
				if !unicode.IsDigit(runes[i]) {
					break
				}
				i++
			}
			tokens = append(tokens, Token{Type: TokenNumber, Value: string(runes[start:i])})
			continue
		}

		if i+1 < len(runes) {
			two := string(runes[i : i+2])
			switch two {
			case "<=", ">=", "==", "!=", "&&", "||":
				tokens = append(tokens, Token{Type: TokenOperator, Value: two})
				i += 2
				continue
			}
		}

		switch r {
		case '+', '-', '*', '/', '<', '>', '!':
			tokens = append(tokens, Token{Type: TokenOperator, Value: string(r)})
		case '.':
			tokens = append(tokens, Token{Type: TokenDot, Value: "."})
		case '(':
			tokens = append(tokens, Token{Type: TokenLParen, Value: "("})
		case ')':
			tokens = append(tokens, Token{Type: TokenRParen, Value: ")"})
		case ',':
			tokens = append(tokens, Token{Type: TokenComma, Value: ","})
		default:
			return nil, fmt.Errorf("unexpected character %q at position %d", r, i)
		}
		i++
	}

	tokens = append(tokens, Token{Type: TokenEOF, Value: ""})
	return tokens, nil
}

func isIdentifierStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentifierPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
