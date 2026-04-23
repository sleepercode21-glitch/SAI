package parser

import "testing"

func TestParseSupportsConnectsList(t *testing.T) {
	program, err := Parse(`app "orders" {
  users 5000
  budget 75usd
}

service api {
  public http
  connects postgres, cache_main
}

database postgres {
  type managed
}

cache cache_main {
  type managed
}`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if got, want := len(program.Resources), 2; got != want {
		t.Fatalf("unexpected resource count: got %d want %d", got, want)
	}

	connects := program.Services[0].Fields[1].Idents
	if got, want := len(connects), 2; got != want {
		t.Fatalf("unexpected connects count: got %d want %d", got, want)
	}
	if got, want := connects[1], "cache_main"; got != want {
		t.Fatalf("unexpected second dependency: got %q want %q", got, want)
	}
}

func TestParseRejectsDuplicateAppDeclaration(t *testing.T) {
	_, err := Parse(`app "a" {}
app "b" {}

service api {}`)
	if err == nil {
		t.Fatal("expected parse to fail for duplicate app declarations")
	}
}
