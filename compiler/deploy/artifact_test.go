package deploy

import (
	"strings"
	"testing"

	buildartifact "github.com/sleepercode/sai/compiler/build"
	infraartifact "github.com/sleepercode/sai/compiler/infra"
)

func TestGenerateAzureBundle(t *testing.T) {
	bundle, err := GenerateBundle(&Plan{
		Cloud:           "azure",
		ApplicationName: "orders",
		Environment:     "prod",
		Region:          "eastus",
		ContextDir:      "server",
		ServiceName:     "api",
		BuildArtifact:   "orders/api",
	}, testInfraPlan("azure"), &infraartifact.Artifact{Format: infraartifact.ArtifactFormatBicep, Content: "targetScope = 'resourceGroup'"},
		&buildartifact.Artifact{Path: "server/Dockerfile", Content: "FROM node:20-alpine"})
	if err != nil {
		t.Fatalf("GenerateBundle returned error: %v", err)
	}

	if got, want := bundle.Provider, "azure"; got != want {
		t.Fatalf("unexpected provider: got %q want %q", got, want)
	}
	if !strings.Contains(bundle.Files["deploy/azure/deploy.sh"], "az deployment group create") {
		t.Fatal("expected azure deploy script to use Azure CLI deployment")
	}
	if !strings.Contains(bundle.Files["deploy/azure/deploy.sh"], "--parameters @deploy/azure/parameters.json") {
		t.Fatal("expected azure deploy script to use generated parameters")
	}
	if !strings.Contains(bundle.Files["deploy/azure/parameters.json"], `"containerImage"`) {
		t.Fatal("expected azure bundle to include parameters file")
	}
	if !strings.Contains(bundle.Files["deploy/azure/deploy.sh"], "az keyvault secret set") || !strings.Contains(bundle.Files["deploy/azure/deploy.sh"], "az appconfig kv set-keyvault") {
		t.Fatal("expected azure deploy script to sync key vault and app config values")
	}
	if !strings.Contains(bundle.Files["deploy/azure/deploy.sh"], `databaseAdminPassword="$DATABASE_ADMIN_PASSWORD"`) {
		t.Fatal("expected azure deploy script to pass database admin password at deploy time")
	}
	if !strings.Contains(bundle.Files["deploy/azure/secrets.env.example"], "SAI_SECRET_POSTGRES_CONNECTION_STRING=") || !strings.Contains(bundle.Files["deploy/azure/secrets.env.example"], "DATABASE_ADMIN_PASSWORD=") {
		t.Fatal("expected azure bundle to include secret seed template")
	}
	if !strings.Contains(bundle.Files[".github/workflows/deploy-azure.yml"], "azure/login@v2") {
		t.Fatal("expected azure workflow to use azure/login")
	}
}

func TestGenerateTerraformBundle(t *testing.T) {
	bundle, err := GenerateBundle(&Plan{
		Cloud:           "aws",
		ApplicationName: "orders",
		Environment:     "prod",
		Region:          "us-east-1",
		ServiceName:     "api",
		ContextDir:      "server",
	}, testInfraPlan("aws"), &infraartifact.Artifact{Format: infraartifact.ArtifactFormatTerraformJSON, Content: `{"terraform":{}}`},
		&buildartifact.Artifact{Path: "server/Dockerfile", Content: "FROM node:20-alpine"})
	if err != nil {
		t.Fatalf("GenerateBundle returned error: %v", err)
	}
	if !strings.Contains(bundle.Files["deploy/terraform/deploy.sh"], "aws ecr get-login-password") {
		t.Fatal("expected aws deploy script to authenticate to ECR")
	}
	if !strings.Contains(bundle.Files["deploy/terraform/deploy.sh"], "aws ecs update-service") {
		t.Fatal("expected aws deploy script to force a new ECS deployment")
	}
	if !strings.Contains(bundle.Files["deploy/terraform/deploy.sh"], "aws secretsmanager put-secret-value") || !strings.Contains(bundle.Files["deploy/terraform/deploy.sh"], "aws ssm put-parameter") {
		t.Fatal("expected aws deploy script to sync secrets manager and ssm values")
	}
	if !strings.Contains(bundle.Files["deploy/terraform/deploy.sh"], `database_admin_password=$DATABASE_ADMIN_PASSWORD`) {
		t.Fatal("expected aws deploy script to pass database admin password at apply time")
	}
	if !strings.Contains(bundle.Files["deploy/terraform/secrets.env.example"], "SAI_SECRET_POSTGRES_CONNECTION_STRING=") || !strings.Contains(bundle.Files["deploy/terraform/secrets.env.example"], "DATABASE_ADMIN_PASSWORD=") {
		t.Fatal("expected aws bundle to include secret seed template")
	}
	if !strings.Contains(bundle.Files["deploy/terraform/terraform.tfvars.json"], `"service_name": "`) {
		t.Fatal("expected terraform bundle to include tfvars")
	}
}

func TestGenerateGCPBundle(t *testing.T) {
	bundle, err := GenerateBundle(&Plan{
		Cloud:           "gcp",
		ApplicationName: "orders",
		Environment:     "prod",
		Region:          "us-central1",
		ServiceName:     "api",
		ContextDir:      "server",
	}, testInfraPlan("gcp"), &infraartifact.Artifact{Format: infraartifact.ArtifactFormatTerraformJSON, Content: `{"terraform":{}}`},
		&buildartifact.Artifact{Path: "server/Dockerfile", Content: "FROM node:20-alpine"})
	if err != nil {
		t.Fatalf("GenerateBundle returned error: %v", err)
	}
	if strings.Contains(bundle.Files["deploy/terraform/deploy.sh"], "aws ecs update-service") {
		t.Fatal("expected gcp deploy script to remain terraform-only")
	}
	if !strings.Contains(bundle.Files["deploy/terraform/deploy.sh"], "gcloud secrets versions add") || !strings.Contains(bundle.Files["deploy/terraform/deploy.sh"], "gcloud auth configure-docker") {
		t.Fatal("expected gcp deploy script to push image and sync secrets")
	}
	if !strings.Contains(bundle.Files["deploy/terraform/secrets.env.example"], "SAI_SECRET_POSTGRES_CONNECTION_STRING=") {
		t.Fatal("expected gcp bundle to include secret seed template")
	}
}

func testInfraPlan(cloud string) *infraartifact.Plan {
	return &infraartifact.Plan{
		ApplicationName: "orders",
		Environment:     "prod",
		Cloud:           cloud,
		Database: &infraartifact.DatabasePlan{
			Name:    "postgres",
			Engine:  "postgres",
			Class:   "small",
			Managed: true,
		},
		AppConfig: infraartifact.AppConfigPlan{
			KeyVaultName: "vault",
			StoreName:    "appconfig",
			Secrets: []infraartifact.AppSecretPlan{
				{Name: "postgres-connection-string", Placeholder: "postgres://app:change-me@postgres:5432/app"},
			},
			Environment: []infraartifact.AppEnvironmentPlan{
				{Name: "PORT", Literal: "3000"},
				{Name: "POSTGRES_URL", SecretRef: "postgres-connection-string"},
			},
		},
	}
}
