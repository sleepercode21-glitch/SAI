package ast

// Position tracks a human-readable source location.
type Position struct {
	Offset int `json:"offset"`
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Span tracks the start and end positions of a syntax node.
type Span struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Program is the root AST node for a .sai document.
type Program struct {
	App       *AppDecl        `json:"app"`
	Services  []*ServiceDecl  `json:"services"`
	Resources []*ResourceDecl `json:"resources"`
	Span      Span            `json:"span"`
}

type AppDecl struct {
	Name   string     `json:"name"`
	Fields []AppField `json:"fields"`
	Span   Span       `json:"span"`
}

type ServiceDecl struct {
	Name   string         `json:"name"`
	Fields []ServiceField `json:"fields"`
	Span   Span           `json:"span"`
}

type ResourceDecl struct {
	Kind   string          `json:"kind"`
	Name   string          `json:"name"`
	Fields []ResourceField `json:"fields"`
	Span   Span            `json:"span"`
}

type AppField struct {
	Kind   string `json:"kind"`
	String string `json:"string,omitempty"`
	Ident  string `json:"ident,omitempty"`
	Int    int    `json:"int,omitempty"`
	Money  int    `json:"money,omitempty"`
	Span   Span   `json:"span"`
}

type ServiceField struct {
	Kind     string   `json:"kind"`
	Ident    string   `json:"ident,omitempty"`
	String   string   `json:"string,omitempty"`
	Int      int      `json:"int,omitempty"`
	Idents   []string `json:"idents,omitempty"`
	Protocol string   `json:"protocol,omitempty"`
	Span     Span     `json:"span"`
}

type ResourceField struct {
	Kind   string `json:"kind"`
	Ident  string `json:"ident,omitempty"`
	String string `json:"string,omitempty"`
	Span   Span   `json:"span"`
}
