package ir

import (
	"testing"

	"github.com/sleepercode/sai/parser"
)

func TestBuildAppliesDefaults(t *testing.T) {
	program, err := parser.Parse(`app "demo" {
  users 1000
  budget 25usd
}

service api {
  port 8080
}`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	result, err := Build(program)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}

	if got, want := result.Application.Cloud, "aws"; got != want {
		t.Fatalf("unexpected cloud default: got %q want %q", got, want)
	}
	if got, want := result.Service.Runtime, "node"; got != want {
		t.Fatalf("unexpected runtime default: got %q want %q", got, want)
	}
	if got, want := result.Service.HealthCheckPath, "/health"; got != want {
		t.Fatalf("unexpected health default: got %q want %q", got, want)
	}
}

func TestBuildRejectsUnknownResourceReference(t *testing.T) {
	program, err := parser.Parse(`app "demo" {
  users 1000
  budget 25usd
}

service api {
  port 8080
  connects postgres
}`)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if _, err := Build(program); err == nil {
		t.Fatal("expected Build to fail for an unknown resource reference")
	}
}
