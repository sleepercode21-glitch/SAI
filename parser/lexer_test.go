package parser

import "testing"

func TestLexerTokenizesManifestWithComments(t *testing.T) {
	lexer := newLexer(`# comment
app "orders" {
  users 5000
  budget 75usd
}
`)

	var tokens []token
	for {
		next, err := lexer.nextToken()
		if err != nil {
			t.Fatalf("nextToken returned error: %v", err)
		}
		tokens = append(tokens, next)
		if next.Type == tokenEOF {
			break
		}
	}

	if len(tokens) < 8 {
		t.Fatalf("expected several tokens, got %d", len(tokens))
	}
	if got, want := tokens[0].Literal, "app"; got != want {
		t.Fatalf("unexpected first token literal: got %q want %q", got, want)
	}
	if got, want := tokens[1].Type, tokenString; got != want {
		t.Fatalf("unexpected second token type: got %q want %q", got, want)
	}

	foundMoney := false
	for _, tok := range tokens {
		if tok.Type == tokenMoney && tok.Literal == "75usd" {
			foundMoney = true
			break
		}
	}
	if !foundMoney {
		t.Fatal("expected lexer output to contain a money token for 75usd")
	}
}

func TestLexerRejectsUnexpectedCharacter(t *testing.T) {
	lexer := newLexer("@")

	if _, err := lexer.nextToken(); err == nil {
		t.Fatal("expected lexer to reject unexpected character")
	}
}
