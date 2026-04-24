package executor

import (
	"context"
	"testing"

	deployartifact "github.com/sleepercode/sai/compiler/deploy"
)

func TestPreflightSupportsKnownProviders(t *testing.T) {
	if _, err := Preflight(context.Background(), &deployartifact.Bundle{Provider: "azure"}); err != nil {
		t.Fatalf("Preflight returned error: %v", err)
	}
}

func TestPreflightRejectsUnknownProvider(t *testing.T) {
	if _, err := Preflight(context.Background(), &deployartifact.Bundle{Provider: "unknown"}); err == nil {
		t.Fatal("expected unknown provider to fail")
	}
}

func TestPreflightAzureDetectsMissingAuthAndBundleFiles(t *testing.T) {
	originalGetenv := getenv
	getenv = func(string) string { return "" }
	defer func() { getenv = originalGetenv }()

	checks, err := Preflight(context.Background(), &deployartifact.Bundle{
		Provider: "azure",
		Files: map[string]string{
			"deploy/azure/deploy.sh": "echo ok",
		},
	})
	if err != nil {
		t.Fatalf("Preflight returned error: %v", err)
	}

	assertCheckStatus(t, checks, "azure-auth", "missing")
	assertCheckStatus(t, checks, "deploy/azure/main.bicep", "missing")
	assertCheckStatus(t, checks, "deploy/azure/deploy.sh", "ok")
}

func TestPreflightGCPAcceptsCredentialEnvironment(t *testing.T) {
	originalGetenv := getenv
	getenv = func(key string) string {
		if key == "GOOGLE_APPLICATION_CREDENTIALS" {
			return "/tmp/key.json"
		}
		return ""
	}
	defer func() { getenv = originalGetenv }()

	checks, err := Preflight(context.Background(), &deployartifact.Bundle{
		Provider: "gcp",
		Files: map[string]string{
			"deploy/terraform/deploy.sh":    "terraform plan",
			"deploy/terraform/main.tf.json": "{}",
		},
	})
	if err != nil {
		t.Fatalf("Preflight returned error: %v", err)
	}

	assertCheckStatus(t, checks, "gcp-auth", "ok")
	assertCheckStatus(t, checks, "deploy/terraform/main.tf.json", "ok")
	assertCheckPresent(t, checks, "gcloud")
	assertCheckPresent(t, checks, "docker")
}

func TestPreflightAWSRequiresDockerAndAWSCLI(t *testing.T) {
	originalGetenv := getenv
	getenv = func(key string) string {
		if key == "AWS_PROFILE" {
			return "default"
		}
		return ""
	}
	defer func() { getenv = originalGetenv }()

	checks, err := Preflight(context.Background(), &deployartifact.Bundle{
		Provider: "aws",
		Files: map[string]string{
			"deploy/terraform/deploy.sh":    "terraform apply",
			"deploy/terraform/main.tf.json": "{}",
		},
	})
	if err != nil {
		t.Fatalf("Preflight returned error: %v", err)
	}
	assertCheckPresent(t, checks, "aws")
	assertCheckPresent(t, checks, "docker")
	assertCheckStatus(t, checks, "aws-auth", "ok")
}

func assertCheckStatus(t *testing.T, checks []PreflightCheck, name string, want string) {
	t.Helper()
	for _, check := range checks {
		if check.Name == name {
			if check.Status != want {
				t.Fatalf("unexpected status for %s: got %q want %q", name, check.Status, want)
			}
			return
		}
	}
	t.Fatalf("missing check named %s", name)
}

func assertCheckPresent(t *testing.T, checks []PreflightCheck, name string) {
	t.Helper()
	for _, check := range checks {
		if check.Name == name {
			return
		}
	}
	t.Fatalf("missing check named %s", name)
}
