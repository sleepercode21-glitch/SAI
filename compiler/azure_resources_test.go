package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanFileAzureCompilesDatabaseCacheAndQueueResources(t *testing.T) {
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

resource azure_openai openai {
  type managed
  size small
}

resource key_vault vault {
  type managed
  size small
}
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	result, err := PlanFile(manifestPath)
	if err != nil {
		t.Fatalf("PlanFile returned error: %v", err)
	}

	if got, want := result.InfraArtifact.Format, "bicep"; string(got) != want {
		t.Fatalf("unexpected infra artifact format: got %q want %q", got, want)
	}
	for _, fragment := range []string{
		"Microsoft.DBforPostgreSQL/flexibleServers",
		"Microsoft.Cache/Redis",
		"Microsoft.ServiceBus/namespaces",
		"Microsoft.CognitiveServices/accounts",
		"Microsoft.KeyVault/vaults",
		"Microsoft.AppConfiguration/configurationStores",
		"keyVaultUrl:",
		"secretRef:",
		"param databaseAdminPassword string",
	} {
		if !strings.Contains(result.InfraArtifact.Content, fragment) {
			t.Fatalf("expected Azure artifact to contain %q", fragment)
		}
	}
	for _, forbidden := range []string{"vaults/secrets@2023-02-01", "change-me-in-keyvault"} {
		if strings.Contains(result.InfraArtifact.Content, forbidden) {
			t.Fatalf("expected Azure artifact to omit %q", forbidden)
		}
	}
}

func TestPlanFileAzureCatalogExampleCompilesCrossCategoryResources(t *testing.T) {
	result, err := PlanFile("../examples/azure-catalog.sai")
	if err != nil {
		t.Fatalf("PlanFile returned error: %v", err)
	}

	if got, want := result.InfraArtifact.Format, "bicep"; string(got) != want {
		t.Fatalf("unexpected infra artifact format: got %q want %q", got, want)
	}

	for _, fragment := range []string{
		"Microsoft.App/containerApps",
		"Microsoft.DBforPostgreSQL/flexibleServers",
		"Microsoft.Cache/Redis",
		"Microsoft.ServiceBus/namespaces",
		"Microsoft.CognitiveServices/accounts",
		"Microsoft.KeyVault/vaults",
		"Microsoft.ContainerRegistry/registries",
		"Microsoft.Network/applicationGateways",
		"Microsoft.Insights/components",
		"Microsoft.MachineLearningServices/workspaces",
		"Microsoft.EventHub/namespaces",
		"Microsoft.DataFactory/factories",
		"Microsoft.Network/azureFirewalls",
		"Microsoft.DesktopVirtualization/hostPools",
		"Microsoft.Devices/IotHubs",
		"Microsoft.AppConfiguration/configurationStores",
		"Microsoft.ApiManagement/service",
		"Microsoft.Synapse/workspaces",
		"Microsoft.Databricks/workspaces",
		"Microsoft.OperationalInsights/workspaces",
		"Microsoft.Authorization/policyAssignments",
		"Microsoft.Cdn/profiles",
		"Microsoft.Network/trafficManagerProfiles",
		"Microsoft.HybridCompute/machines",
		"Microsoft.BotService/botServices",
		"Microsoft.DesktopVirtualization/hostPools",
		"Microsoft.DocumentDB/databaseAccounts",
		"Microsoft.Sql/servers/databases",
		"Microsoft.DBforMySQL/flexibleServers",
		"Microsoft.DBforMariaDB/servers",
		"Microsoft.Storage/storageAccounts",
		"Microsoft.EventGrid/topics",
		"Microsoft.Logic/workflows",
		"Microsoft.Security/pricings",
		"Microsoft.OperationalInsights/workspaces/providers/onboardingStates",
		"Microsoft.ManagedIdentity/userAssignedIdentities",
		"Microsoft.Network/privateEndpoints",
		"Microsoft.Network/ddosProtectionPlans",
		"Microsoft.AzureStackHCI/clusters",
		"Microsoft.IoTCentral/iotApps",
		"Microsoft.Intune/locations",
	} {
		if !strings.Contains(result.InfraArtifact.Content, fragment) {
			t.Fatalf("expected Azure artifact to contain %q", fragment)
		}
	}
}
