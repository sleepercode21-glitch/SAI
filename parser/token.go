package parser

import "github.com/sleepercode/sai/ast"

type tokenType string

const (
	tokenEOF      tokenType = "EOF"
	tokenIdent    tokenType = "IDENT"
	tokenString   tokenType = "STRING"
	tokenInteger  tokenType = "INTEGER"
	tokenMoney    tokenType = "MONEY"
	tokenLBrace   tokenType = "{"
	tokenRBrace   tokenType = "}"
	tokenComma    tokenType = ","
)

type token struct {
	Type    tokenType
	Literal string
	Int     int
	Span    ast.Span
}
