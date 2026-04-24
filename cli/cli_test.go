package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sleepercode/sai/executor"
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
  connects postgres, cache, jobs
}

database postgres {
  type managed
  size small
}

cache cache {
  type managed
  size small
}

queue jobs {
  type managed
  size small
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
	if !strings.Contains(output, "Microsoft.Cache/Redis") || !strings.Contains(output, "Microsoft.ServiceBus/namespaces") {
		t.Fatalf("expected Azure cache and queue resources in output, got %q", output)
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

func TestDeployCommandWritesAzureBundle(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "azure.sai")
	outputDir := filepath.Join(tempDir, "out")
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
}
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return NewDeployCommand().Run([]string{"--path", manifestPath, "--output-dir", outputDir})
	})
	if err != nil {
		t.Fatalf("deploy command returned error: %v", err)
	}
	if !strings.Contains(output, "wrote deploy bundle") {
		t.Fatalf("expected deploy output to mention bundle, got %q", output)
	}
	for _, expected := range []string{
		filepath.Join(outputDir, "deploy/azure/main.bicep"),
		filepath.Join(outputDir, "deploy/azure/parameters.json"),
		filepath.Join(outputDir, "deploy/azure/deploy.sh"),
		filepath.Join(outputDir, ".github/workflows/deploy-azure.yml"),
	} {
		if _, err := os.Stat(expected); err != nil {
			t.Fatalf("expected deploy bundle file %s to exist: %v", expected, err)
		}
	}
}

func TestDeployCommandWritesTerraformBundleForGCP(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "gcp.sai")
	outputDir := filepath.Join(tempDir, "out")
	manifest := `app "orders" {
  cloud gcp
  region "us-central1"
  users 5000
  budget 75usd
}

service api {
  runtime node
  path "server"
  port 3000
  public http
}
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return NewDeployCommand().Run([]string{"--path", manifestPath, "--output-dir", outputDir})
	})
	if err != nil {
		t.Fatalf("deploy command returned error: %v", err)
	}
	if !strings.Contains(output, "wrote deploy bundle") {
		t.Fatalf("expected deploy output to mention bundle, got %q", output)
	}
	for _, expected := range []string{
		filepath.Join(outputDir, "deploy/terraform/main.tf.json"),
		filepath.Join(outputDir, "deploy/terraform/terraform.tfvars.json"),
		filepath.Join(outputDir, "deploy/terraform/deploy.sh"),
	} {
		if _, err := os.Stat(expected); err != nil {
			t.Fatalf("expected deploy bundle file %s to exist: %v", expected, err)
		}
	}
}

func TestLogsCommandReadsLatestReleaseLog(t *testing.T) {
	tempDir := t.TempDir()
	if err := executor.EnsureStateLayout(tempDir); err != nil {
		t.Fatalf("EnsureStateLayout returned error: %v", err)
	}
	record := &executor.ReleaseRecord{
		ID:         "20260423T000000Z",
		Provider:   "azure",
		BundleRoot: filepath.Join(tempDir, "bundle"),
		Status:     "succeeded",
		Operation:  "deploy",
		LogPath:    filepath.Join(tempDir, executor.StateDirName, executor.LogsDirName, "20260423T000000Z.log"),
		Events: []executor.ExecutionEvent{
			{Name: "plan-commands", Status: "ok"},
		},
	}
	if err := os.WriteFile(record.LogPath, []byte("deployment log output\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}
	if err := executor.SaveRelease(tempDir, record); err != nil {
		t.Fatalf("SaveRelease returned error: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return NewLogsCommand().Run([]string{"--state-root", tempDir})
	})
	if err != nil {
		t.Fatalf("logs command returned error: %v", err)
	}
	if !strings.Contains(output, "deployment log output") {
		t.Fatalf("expected logs output, got %q", output)
	}
	if !strings.Contains(output, "release=20260423T000000Z") || !strings.Contains(output, "plan-commands: ok") {
		t.Fatalf("expected release summary and event output, got %q", output)
	}
}

func TestRollbackCommandPrintsCurrentTarget(t *testing.T) {
	tempDir := t.TempDir()
	if err := executor.EnsureStateLayout(tempDir); err != nil {
		t.Fatalf("EnsureStateLayout returned error: %v", err)
	}
	record := &executor.ReleaseRecord{
		ID:         "20260423T000001Z",
		Provider:   "azure",
		BundleRoot: filepath.Join(tempDir, "bundle"),
		Status:     "succeeded",
		LogPath:    filepath.Join(tempDir, executor.StateDirName, executor.LogsDirName, "20260423T000001Z.log"),
	}
	if err := executor.SaveRelease(tempDir, record); err != nil {
		t.Fatalf("SaveRelease returned error: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return NewRollbackCommand().Run([]string{"--state-root", tempDir})
	})
	if err != nil {
		t.Fatalf("rollback command returned error: %v", err)
	}
	if !strings.Contains(output, "rollback target=") {
		t.Fatalf("expected rollback output, got %q", output)
	}
}

func TestRollbackCommandExecutesStoredBundle(t *testing.T) {
	tempDir := t.TempDir()
	if err := executor.EnsureStateLayout(tempDir); err != nil {
		t.Fatalf("EnsureStateLayout returned error: %v", err)
	}
	record := &executor.ReleaseRecord{
		ID:         "20260423T000003Z",
		Provider:   "azure",
		BundleRoot: filepath.Join(tempDir, "bundle"),
		Status:     "succeeded",
		LogPath:    filepath.Join(tempDir, executor.StateDirName, executor.LogsDirName, "20260423T000003Z.log"),
		BundleFiles: map[string]string{
			"deploy/azure/deploy.sh": "#!/usr/bin/env bash\nset -euo pipefail\necho rollback\n",
		},
	}
	if err := executor.SaveRelease(tempDir, record); err != nil {
		t.Fatalf("SaveRelease returned error: %v", err)
	}

	output, err := captureStdout(t, func() error {
		return NewRollbackCommand().Run([]string{"--state-root", tempDir, "--execute", "--preflight=false"})
	})
	if err != nil {
		t.Fatalf("rollback command returned error: %v", err)
	}
	if !strings.Contains(output, "rollback executed") {
		t.Fatalf("expected rollback execution output, got %q", output)
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
