package parser

import (
	"fmt"

	"github.com/sleepercode/sai/ast"
)

type Parser struct {
	lexer    *lexer
	current  token
	previous token
}

func Parse(input string) (*ast.Program, error) {
	p := &Parser{lexer: newLexer(input)}
	if err := p.advance(); err != nil {
		return nil, err
	}
	return p.parseProgram()
}

func (p *Parser) parseProgram() (*ast.Program, error) {
	program := &ast.Program{}
	program.Span.Start = p.current.Span.Start

	for p.current.Type != tokenEOF {
		switch p.current.Literal {
		case "app":
			if program.App != nil {
				return nil, p.errorf("multiple app declarations are not allowed", p.current.Span)
			}
			appDecl, err := p.parseApp()
			if err != nil {
				return nil, err
			}
			program.App = appDecl
		case "service":
			serviceDecl, err := p.parseService()
			if err != nil {
				return nil, err
			}
			program.Services = append(program.Services, serviceDecl)
		case "database", "cache", "queue":
			resourceDecl, err := p.parseResource()
			if err != nil {
				return nil, err
			}
			program.Resources = append(program.Resources, resourceDecl)
		default:
			return nil, p.errorf(fmt.Sprintf("unexpected top-level declaration %q", p.current.Literal), p.current.Span)
		}
	}

	program.Span.End = p.current.Span.End
	return program, nil
}

func (p *Parser) parseApp() (*ast.AppDecl, error) {
	start := p.current.Span.Start
	if err := p.expectLiteral("app"); err != nil {
		return nil, err
	}
	name, err := p.expectString()
	if err != nil {
		return nil, err
	}
	if err := p.expectType(tokenLBrace, "{"); err != nil {
		return nil, err
	}

	fields := []ast.AppField{}
	for p.current.Type != tokenRBrace {
		field, err := p.parseAppField()
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	end := p.current.Span.End
	if err := p.expectType(tokenRBrace, "}"); err != nil {
		return nil, err
	}

	return &ast.AppDecl{
		Name:   name,
		Fields: fields,
		Span:   ast.Span{Start: start, End: end},
	}, nil
}

func (p *Parser) parseService() (*ast.ServiceDecl, error) {
	start := p.current.Span.Start
	if err := p.expectLiteral("service"); err != nil {
		return nil, err
	}
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	if err := p.expectType(tokenLBrace, "{"); err != nil {
		return nil, err
	}

	fields := []ast.ServiceField{}
	for p.current.Type != tokenRBrace {
		field, err := p.parseServiceField()
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	end := p.current.Span.End
	if err := p.expectType(tokenRBrace, "}"); err != nil {
		return nil, err
	}

	return &ast.ServiceDecl{
		Name:   name,
		Fields: fields,
		Span:   ast.Span{Start: start, End: end},
	}, nil
}

func (p *Parser) parseResource() (*ast.ResourceDecl, error) {
	kind := p.current.Literal
	start := p.current.Span.Start
	if err := p.advance(); err != nil {
		return nil, err
	}
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	if err := p.expectType(tokenLBrace, "{"); err != nil {
		return nil, err
	}

	fields := []ast.ResourceField{}
	for p.current.Type != tokenRBrace {
		field, err := p.parseResourceField()
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}

	end := p.current.Span.End
	if err := p.expectType(tokenRBrace, "}"); err != nil {
		return nil, err
	}

	return &ast.ResourceDecl{
		Kind:   kind,
		Name:   name,
		Fields: fields,
		Span:   ast.Span{Start: start, End: end},
	}, nil
}

func (p *Parser) parseAppField() (ast.AppField, error) {
	kind := p.current.Literal
	spanStart := p.current.Span.Start
	switch kind {
	case "cloud", "env":
		if err := p.advance(); err != nil {
			return ast.AppField{}, err
		}
		value, err := p.expectIdent()
		if err != nil {
			return ast.AppField{}, err
		}
		return ast.AppField{Kind: kind, Ident: value, Span: ast.Span{Start: spanStart, End: p.previousSpan().End}}, nil
	case "region":
		if err := p.advance(); err != nil {
			return ast.AppField{}, err
		}
		value, err := p.expectString()
		if err != nil {
			return ast.AppField{}, err
		}
		return ast.AppField{Kind: kind, String: value, Span: ast.Span{Start: spanStart, End: p.previousSpan().End}}, nil
	case "users":
		if err := p.advance(); err != nil {
			return ast.AppField{}, err
		}
		value, err := p.expectInteger()
		if err != nil {
			return ast.AppField{}, err
		}
		return ast.AppField{Kind: kind, Int: value, Span: ast.Span{Start: spanStart, End: p.previousSpan().End}}, nil
	case "budget":
		if err := p.advance(); err != nil {
			return ast.AppField{}, err
		}
		value, err := p.expectMoney()
		if err != nil {
			return ast.AppField{}, err
		}
		return ast.AppField{Kind: kind, Money: value, Span: ast.Span{Start: spanStart, End: p.previousSpan().End}}, nil
	default:
		return ast.AppField{}, p.errorf(fmt.Sprintf("unknown app field %q", kind), p.current.Span)
	}
}

func (p *Parser) parseServiceField() (ast.ServiceField, error) {
	kind := p.current.Literal
	spanStart := p.current.Span.Start
	switch kind {
	case "runtime", "scale":
		if err := p.advance(); err != nil {
			return ast.ServiceField{}, err
		}
		value, err := p.expectIdent()
		if err != nil {
			return ast.ServiceField{}, err
		}
		return ast.ServiceField{Kind: kind, Ident: value, Span: ast.Span{Start: spanStart, End: p.previousSpan().End}}, nil
	case "path", "health":
		if err := p.advance(); err != nil {
			return ast.ServiceField{}, err
		}
		value, err := p.expectString()
		if err != nil {
			return ast.ServiceField{}, err
		}
		return ast.ServiceField{Kind: kind, String: value, Span: ast.Span{Start: spanStart, End: p.previousSpan().End}}, nil
	case "port":
		if err := p.advance(); err != nil {
			return ast.ServiceField{}, err
		}
		value, err := p.expectInteger()
		if err != nil {
			return ast.ServiceField{}, err
		}
		return ast.ServiceField{Kind: kind, Int: value, Span: ast.Span{Start: spanStart, End: p.previousSpan().End}}, nil
	case "public":
		if err := p.advance(); err != nil {
			return ast.ServiceField{}, err
		}
		protocol, err := p.expectIdent()
		if err != nil {
			return ast.ServiceField{}, err
		}
		return ast.ServiceField{Kind: kind, Protocol: protocol, Span: ast.Span{Start: spanStart, End: p.previousSpan().End}}, nil
	case "private":
		if err := p.advance(); err != nil {
			return ast.ServiceField{}, err
		}
		return ast.ServiceField{Kind: kind, Span: ast.Span{Start: spanStart, End: p.previousSpan().End}}, nil
	case "connects":
		if err := p.advance(); err != nil {
			return ast.ServiceField{}, err
		}
		idents, err := p.parseIdentList()
		if err != nil {
			return ast.ServiceField{}, err
		}
		return ast.ServiceField{Kind: kind, Idents: idents, Span: ast.Span{Start: spanStart, End: p.previousSpan().End}}, nil
	default:
		return ast.ServiceField{}, p.errorf(fmt.Sprintf("unknown service field %q", kind), p.current.Span)
	}
}

func (p *Parser) parseResourceField() (ast.ResourceField, error) {
	kind := p.current.Literal
	spanStart := p.current.Span.Start
	switch kind {
	case "type", "size":
		if err := p.advance(); err != nil {
			return ast.ResourceField{}, err
		}
		value, err := p.expectIdent()
		if err != nil {
			return ast.ResourceField{}, err
		}
		return ast.ResourceField{Kind: kind, Ident: value, Span: ast.Span{Start: spanStart, End: p.previousSpan().End}}, nil
	default:
		return ast.ResourceField{}, p.errorf(fmt.Sprintf("unknown resource field %q", kind), p.current.Span)
	}
}

func (p *Parser) parseIdentList() ([]string, error) {
	values := []string{}
	first, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	values = append(values, first)

	for p.current.Type == tokenComma {
		if err := p.advance(); err != nil {
			return nil, err
		}
		next, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		values = append(values, next)
	}

	return values, nil
}

func (p *Parser) expectLiteral(value string) error {
	if p.current.Literal != value {
		return p.errorf(fmt.Sprintf("expected %q", value), p.current.Span)
	}
	return p.advance()
}

func (p *Parser) expectType(tt tokenType, label string) error {
	if p.current.Type != tt {
		return p.errorf(fmt.Sprintf("expected %s", label), p.current.Span)
	}
	return p.advance()
}

func (p *Parser) expectString() (string, error) {
	if p.current.Type != tokenString {
		return "", p.errorf("expected string literal", p.current.Span)
	}
	value := p.current.Literal
	return value, p.advanceValue()
}

func (p *Parser) expectIdent() (string, error) {
	if p.current.Type != tokenIdent {
		return "", p.errorf("expected identifier", p.current.Span)
	}
	value := p.current.Literal
	return value, p.advanceValue()
}

func (p *Parser) expectInteger() (int, error) {
	if p.current.Type != tokenInteger {
		return 0, p.errorf("expected integer", p.current.Span)
	}
	value := p.current.Int
	return value, p.advanceValue()
}

func (p *Parser) expectMoney() (int, error) {
	if p.current.Type != tokenMoney {
		return 0, p.errorf("expected money value like 75usd", p.current.Span)
	}
	value := p.current.Int
	return value, p.advanceValue()
}

func (p *Parser) advanceValue() error {
	return p.advance()
}

func (p *Parser) rememberCurrent() {
	p.previous = p.current
}

func (p *Parser) previousSpan() ast.Span {
	return p.previous.Span
}

func (p *Parser) advance() error {
	p.rememberCurrent()
	next, err := p.lexer.nextToken()
	if err != nil {
		return err
	}
	p.current = next
	return nil
}

func (p *Parser) errorf(message string, span ast.Span) error {
	return fmt.Errorf("parse error at %d:%d: %s", span.Start.Line, span.Start.Column, message)
}
