package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanFileRejectsAzureResourceOnAWS(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "aws-invalid.sai")
	manifest := `app "orders" {
  cloud aws
  region "us-east-1"
  users 5000
  budget 75usd
}

service api {
  runtime node
  path "server"
  port 3000
  public http
}

resource azure_openai openai {
  type managed
  size small
}
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	_, err := PlanFile(manifestPath)
	if err == nil || !strings.Contains(err.Error(), `aws backend does not support resource kind "azure_openai"`) {
		t.Fatalf("expected aws resource validation error, got %v", err)
	}
}

func TestPlanFileRejectsInvalidHealthPath(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "invalid-health.sai")
	manifest := `app "orders" {
  cloud azure
  region "eastus"
  users 5000
  budget 75usd
}

service api {
  runtime node
  path "server"
  port 3000
  public http
  health "health"
}
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	_, err := PlanFile(manifestPath)
	if err == nil || !strings.Contains(err.Error(), `health check path "health" must start with /`) {
		t.Fatalf("expected invalid health path validation error, got %v", err)
	}
}
