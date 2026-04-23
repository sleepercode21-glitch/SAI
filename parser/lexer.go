package parser

import (
	"fmt"
	"strconv"
	"unicode"

	"github.com/sleepercode/sai/ast"
)

type lexer struct {
	input  []rune
	offset int
	line   int
	column int
}

func newLexer(input string) *lexer {
	return &lexer{
		input:  []rune(input),
		line:   1,
		column: 1,
	}
}

func (l *lexer) nextToken() (token, error) {
	l.skipWhitespace()

	start := l.position()
	if l.isEOF() {
		return token{
			Type: tokenEOF,
			Span: ast.Span{Start: start, End: start},
		}, nil
	}

	ch := l.peek()
	switch {
	case ch == '{':
		l.advance()
		return token{Type: tokenLBrace, Literal: "{", Span: ast.Span{Start: start, End: l.position()}}, nil
	case ch == '}':
		l.advance()
		return token{Type: tokenRBrace, Literal: "}", Span: ast.Span{Start: start, End: l.position()}}, nil
	case ch == ',':
		l.advance()
		return token{Type: tokenComma, Literal: ",", Span: ast.Span{Start: start, End: l.position()}}, nil
	case ch == '"':
		return l.readString()
	case unicode.IsDigit(ch):
		return l.readNumberLike()
	case isIdentifierStart(ch):
		return l.readIdentifier()
	default:
		return token{}, fmt.Errorf("unexpected character %q at %d:%d", ch, start.Line, start.Column)
	}
}

func (l *lexer) readString() (token, error) {
	start := l.position()
	l.advance()

	var value []rune
	for !l.isEOF() && l.peek() != '"' {
		value = append(value, l.peek())
		l.advance()
	}

	if l.isEOF() {
		return token{}, fmt.Errorf("unterminated string at %d:%d", start.Line, start.Column)
	}

	l.advance()
	return token{
		Type:    tokenString,
		Literal: string(value),
		Span:    ast.Span{Start: start, End: l.position()},
	}, nil
}

func (l *lexer) readNumberLike() (token, error) {
	start := l.position()

	var digits []rune
	for !l.isEOF() && unicode.IsDigit(l.peek()) {
		digits = append(digits, l.peek())
		l.advance()
	}

	number, err := strconv.Atoi(string(digits))
	if err != nil {
		return token{}, err
	}

	if l.peekSequence("usd") {
		l.advance()
		l.advance()
		l.advance()
		return token{
			Type:    tokenMoney,
			Literal: string(digits) + "usd",
			Int:     number,
			Span:    ast.Span{Start: start, End: l.position()},
		}, nil
	}

	return token{
		Type:    tokenInteger,
		Literal: string(digits),
		Int:     number,
		Span:    ast.Span{Start: start, End: l.position()},
	}, nil
}

func (l *lexer) readIdentifier() (token, error) {
	start := l.position()
	var value []rune
	for !l.isEOF() && isIdentifierPart(l.peek()) {
		value = append(value, l.peek())
		l.advance()
	}
	return token{
		Type:    tokenIdent,
		Literal: string(value),
		Span:    ast.Span{Start: start, End: l.position()},
	}, nil
}

func (l *lexer) skipWhitespace() {
	for !l.isEOF() {
		ch := l.peek()
		if ch == '#' {
			for !l.isEOF() && l.peek() != '\n' {
				l.advance()
			}
			continue
		}
		if !unicode.IsSpace(ch) {
			return
		}
		l.advance()
	}
}

func (l *lexer) peekSequence(value string) bool {
	if l.offset+len([]rune(value)) > len(l.input) {
		return false
	}
	for i, r := range value {
		if l.input[l.offset+i] != r {
			return false
		}
	}
	return true
}

func (l *lexer) peek() rune {
	if l.isEOF() {
		return 0
	}
	return l.input[l.offset]
}

func (l *lexer) advance() {
	if l.isEOF() {
		return
	}
	ch := l.input[l.offset]
	l.offset++
	if ch == '\n' {
		l.line++
		l.column = 1
		return
	}
	l.column++
}

func (l *lexer) position() ast.Position {
	return ast.Position{
		Offset: l.offset,
		Line:   l.line,
		Column: l.column,
	}
}

func (l *lexer) isEOF() bool {
	return l.offset >= len(l.input)
}

func isIdentifierStart(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

func isIdentifierPart(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-'
}
