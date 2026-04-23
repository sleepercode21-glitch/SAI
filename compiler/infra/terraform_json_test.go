package infra

import (
	"strings"
	"testing"

	"github.com/sleepercode/sai/ir"
	"github.com/sleepercode/sai/planner"
)

func TestGenerateArtifactUsesAWSBackend(t *testing.T) {
	artifact, err := GenerateArtifact(testPlan("aws", "us-east-1"))
	if err != nil {
		t.Fatalf("GenerateArtifact returned error: %v", err)
	}
	if got, want := artifact.Format, ArtifactFormatTerraformJSON; got != want {
		t.Fatalf("unexpected artifact format: got %q want %q", got, want)
	}
	for _, fragment := range []string{`"aws_vpc"`, `"aws_ecs_service"`, `"aws_db_instance"`} {
		if !strings.Contains(artifact.Content, fragment) {
			t.Fatalf("expected AWS artifact to contain %q", fragment)
		}
	}
}

func TestGenerateArtifactUsesAzureBackend(t *testing.T) {
	artifact, err := GenerateArtifact(testPlan("azure", "eastus"))
	if err != nil {
		t.Fatalf("GenerateArtifact returned error: %v", err)
	}
	if got, want := artifact.Format, ArtifactFormatBicep; got != want {
		t.Fatalf("unexpected artifact format: got %q want %q", got, want)
	}
	for _, fragment := range []string{"targetScope = 'resourceGroup'", "Microsoft.App/containerApps", "Microsoft.DBforPostgreSQL/flexibleServers"} {
		if !strings.Contains(artifact.Content, fragment) {
			t.Fatalf("expected Azure artifact to contain %q", fragment)
		}
	}
}

func TestGenerateArtifactUsesGCPBackend(t *testing.T) {
	artifact, err := GenerateArtifact(testPlan("gcp", "us-central1"))
	if err != nil {
		t.Fatalf("GenerateArtifact returned error: %v", err)
	}
	if got, want := artifact.Format, ArtifactFormatTerraformJSON; got != want {
		t.Fatalf("unexpected artifact format: got %q want %q", got, want)
	}
	for _, fragment := range []string{`"google_compute_network"`, `"google_cloud_run_v2_service"`, `"google_sql_database_instance"`} {
		if !strings.Contains(artifact.Content, fragment) {
			t.Fatalf("expected GCP artifact to contain %q", fragment)
		}
	}
}

func TestGenerateTerraformJSONSupportsGCP(t *testing.T) {
	data, err := GenerateTerraformJSON(testPlan("gcp", "us-central1"))
	if err != nil {
		t.Fatalf("GenerateTerraformJSON returned error: %v", err)
	}
	for _, fragment := range []string{`"google_compute_network"`, `"google_cloud_run_v2_service"`} {
		if !strings.Contains(string(data), fragment) {
			t.Fatalf("expected GCP terraform JSON to contain %q", fragment)
		}
	}
}

func TestGenerateTerraformJSONRejectsAzure(t *testing.T) {
	_, err := GenerateTerraformJSON(testPlan("azure", "eastus"))
	if err == nil {
		t.Fatal("expected terraform json generation to fail for azure")
	}
}

func TestGenerateArtifactRejectsUnsupportedCloud(t *testing.T) {
	_, err := GenerateArtifact(&Plan{Cloud: "digitalocean"})
	if err == nil {
		t.Fatal("expected unsupported cloud to fail")
	}
}

func testPlan(cloud, region string) *Plan {
	program := &ir.ProgramIR{
		Application: ir.ApplicationIR{
			Name:      "orders",
			Cloud:     cloud,
			Region:    region,
			BudgetUSD: 75,
			Env:       "prod",
		},
		Service: ir.ServiceIR{
			Name:            "api",
			Runtime:         "node",
			Port:            3000,
			Exposure:        ir.ExposurePublicHTTP,
			HealthCheckPath: "/health",
			Connects:        []string{"postgres"},
		},
		Resources: []ir.ResourceIR{
			{Name: "postgres", Kind: "database", Type: "managed", Size: "small"},
		},
	}
	deployment := &planner.Plan{
		Profile:      planner.ProfileBalancedWeb,
		InfraClass:   "managed-small",
		MinInstances: 1,
		MaxInstances: 3,
		EstimatedUSD: 75,
	}
	plan, _ := Lower(program, deployment)
	return plan
}
