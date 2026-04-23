package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitValidateAndPlanCommands(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "sai.sai")

	initOutput, err := captureStdout(t, func() error {
		return NewInitCommand().Run([]string{"--path", manifestPath})
	})
	if err != nil {
		t.Fatalf("init command returned error: %v", err)
	}
	if !strings.Contains(initOutput, "created") {
		t.Fatalf("expected init output to mention created, got %q", initOutput)
	}

	validateOutput, err := captureStdout(t, func() error {
		return NewValidateCommand().Run([]string{"--path", manifestPath})
	})
	if err != nil {
		t.Fatalf("validate command returned error: %v", err)
	}
	if !strings.Contains(validateOutput, "is valid") {
		t.Fatalf("expected validate output to confirm validity, got %q", validateOutput)
	}

	planOutput, err := captureStdout(t, func() error {
		return NewPlanCommand().Run([]string{"--path", manifestPath, "--terraform-json"})
	})
	if err != nil {
		t.Fatalf("plan command returned error: %v", err)
	}
	if !strings.Contains(planOutput, `"aws_ecs_service"`) {
		t.Fatalf("expected terraform JSON in plan output, got %q", planOutput)
	}
}

func TestPlanCommandEmitsAzureInfraArtifact(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "azure.sai")
	manifest := `app "orders" {
  cloud azure
  region "eastus"
  users 5000
  budget 75usd
}

service api {
  port 3000
  public http
}
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return NewPlanCommand().Run([]string{"--path", manifestPath, "--infra-artifact"})
	})
	if err != nil {
		t.Fatalf("plan command returned error: %v", err)
	}
	if !strings.Contains(output, "Microsoft.App/containerApps") {
		t.Fatalf("expected Azure Bicep in output, got %q", output)
	}
}

func TestPlanCommandRejectsTerraformJSONForAzure(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "azure.sai")
	manifest := `app "orders" {
  cloud azure
  region "eastus"
  users 5000
  budget 75usd
}

service api {
  port 3000
  public http
}
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	err := NewPlanCommand().Run([]string{"--path", manifestPath, "--terraform-json"})
	if err == nil {
		t.Fatal("expected terraform-json flag to fail for azure")
	}
}

func TestPlanCommandEmitsGCPInfraArtifact(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "gcp.sai")
	manifest := `app "orders" {
  cloud gcp
  region "us-central1"
  users 5000
  budget 75usd
}

service api {
  port 3000
  public http
}
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return NewPlanCommand().Run([]string{"--path", manifestPath, "--infra-artifact"})
	})
	if err != nil {
		t.Fatalf("plan command returned error: %v", err)
	}
	if !strings.Contains(output, `"google_cloud_run_v2_service"`) {
		t.Fatalf("expected GCP artifact in output, got %q", output)
	}
}

func TestPlanCommandEmitsGCPAsTerraformJSON(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "gcp.sai")
	manifest := `app "orders" {
  cloud gcp
  region "us-central1"
  users 5000
  budget 75usd
}

service api {
  port 3000
  public http
}
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return NewPlanCommand().Run([]string{"--path", manifestPath, "--terraform-json"})
	})
	if err != nil {
		t.Fatalf("plan command returned error: %v", err)
	}
	if !strings.Contains(output, `"google_compute_network"`) {
		t.Fatalf("expected GCP terraform JSON in output, got %q", output)
	}
}

func TestAppUnknownCommandFails(t *testing.T) {
	err := NewApp().Run([]string{"unknown"})
	if err == nil {
		t.Fatal("expected unknown command to fail")
	}
}

func TestHelpListsCommands(t *testing.T) {
	output, err := captureStdout(t, func() error {
		return NewApp().Run(nil)
	})
	if err == nil {
		t.Fatal("expected help to return an error when no command is provided")
	}
	if !strings.Contains(output, "validate") || !strings.Contains(output, "rollback") {
		t.Fatalf("expected help output to list commands, got %q", output)
	}
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe returned error: %v", err)
	}

	os.Stdout = writer
	runErr := fn()
	_ = writer.Close()
	os.Stdout = originalStdout

	var buffer bytes.Buffer
	if _, err := io.Copy(&buffer, reader); err != nil {
		t.Fatalf("io.Copy returned error: %v", err)
	}
	_ = reader.Close()

	return buffer.String(), runErr
}
