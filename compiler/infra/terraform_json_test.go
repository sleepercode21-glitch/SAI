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
	for _, fragment := range []string{`"aws_vpc"`, `"aws_ecs_service"`, `"aws_db_instance"`, `"aws_ecr_repository"`, `"aws_lb"`, `"aws_iam_role"`, `"aws_elasticache_cluster"`, `"aws_sqs_queue"`, `"aws_secretsmanager_secret"`, `"aws_ssm_parameter"`, `"database_admin_password"`} {
		if !strings.Contains(artifact.Content, fragment) {
			t.Fatalf("expected AWS artifact to contain %q", fragment)
		}
	}
	for _, forbidden := range []string{`"aws_secretsmanager_secret_version"`, `change-me-in-secrets`, `secret_string`} {
		if strings.Contains(artifact.Content, forbidden) {
			t.Fatalf("expected AWS artifact to omit %q", forbidden)
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
	for _, fragment := range []string{"targetScope = 'resourceGroup'", "param containerImage string", "keyVaultUrl:", "secretRef:", "Microsoft.App/containerApps", "Microsoft.OperationalInsights/workspaces", "Microsoft.ManagedIdentity/userAssignedIdentities", "Microsoft.AppConfiguration/configurationStores", "Microsoft.DBforPostgreSQL/flexibleServers", "Microsoft.Cache/Redis", "Microsoft.ServiceBus/namespaces", "Microsoft.CognitiveServices/accounts", "Microsoft.KeyVault/vaults"} {
		if !strings.Contains(artifact.Content, fragment) {
			t.Fatalf("expected Azure artifact to contain %q", fragment)
		}
	}
	for _, fragment := range []string{"@secure()", "param databaseAdminPassword string"} {
		if !strings.Contains(artifact.Content, fragment) {
			t.Fatalf("expected Azure artifact to contain %q", fragment)
		}
	}
	for _, forbidden := range []string{"vaults/secrets@2023-02-01", "change-me-in-keyvault"} {
		if strings.Contains(artifact.Content, forbidden) {
			t.Fatalf("expected Azure artifact to omit %q", forbidden)
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
	for _, fragment := range []string{`"google_compute_network"`, `"google_cloud_run_v2_service"`, `"google_sql_database_instance"`, `"google_secret_manager_secret"`, `"google_service_account"`, `"value_source"`} {
		if !strings.Contains(artifact.Content, fragment) {
			t.Fatalf("expected GCP artifact to contain %q", fragment)
		}
	}
	for _, forbidden := range []string{`"google_secret_manager_secret_version"`, `secret_data`} {
		if strings.Contains(artifact.Content, forbidden) {
			t.Fatalf("expected GCP artifact to omit %q", forbidden)
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
			{Name: "cache", Kind: "cache", Type: "managed", Size: "small"},
			{Name: "jobs", Kind: "queue", Type: "managed", Size: "small"},
			{Name: "openai", Kind: "azure_openai", Type: "managed", Size: "small"},
			{Name: "vault", Kind: "key_vault", Type: "managed", Size: "small"},
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
