package infra

import (
	"fmt"
	"strings"
)

func generateAzureBicep(plan *Plan) (string, error) {
	stack := slugify(fmt.Sprintf("%s-%s", plan.ApplicationName, plan.Environment))
	var builder strings.Builder
	builder.WriteString("targetScope = 'resourceGroup'\n\n")
	builder.WriteString(fmt.Sprintf("param location string = '%s'\n", plan.Region))
	builder.WriteString(fmt.Sprintf("param appName string = '%s'\n", slugify(plan.ApplicationName)))
	builder.WriteString(fmt.Sprintf("param environmentName string = '%s'\n", slugify(plan.Environment)))
	builder.WriteString(fmt.Sprintf("param serviceName string = '%s'\n", slugify(plan.Compute.ServiceName)))
	builder.WriteString(fmt.Sprintf("param containerImage string = '%s.azurecr.io/%s/%s:latest'\n", slugify(plan.ApplicationName), slugify(plan.ApplicationName), plan.Compute.ServiceName))
	builder.WriteString(fmt.Sprintf("param containerPort int = %d\n", plan.Network.ServicePort))
	builder.WriteString(fmt.Sprintf("param minReplicas int = %d\n", plan.Compute.MinInstances))
	builder.WriteString(fmt.Sprintf("param maxReplicas int = %d\n", plan.Compute.MaxInstances))
	if plan.Database != nil && plan.Database.Managed {
		builder.WriteString("@secure()\n")
		builder.WriteString("param databaseAdminPassword string\n")
	}
	builder.WriteString("param tags object = {\n")
	builder.WriteString(fmt.Sprintf("  app: '%s'\n", plan.ApplicationName))
	builder.WriteString(fmt.Sprintf("  env: '%s'\n", plan.Environment))
	builder.WriteString("  managedBy: 'sai'\n")
	builder.WriteString("}\n\n")
	builder.WriteString(fmt.Sprintf("var stackName = '%s'\n", stack))
	builder.WriteString("var containerAppName = '${stackName}-${serviceName}'\n\n")
	builder.WriteString("resource workspace 'Microsoft.OperationalInsights/workspaces@2022-10-01' = {\n")
	builder.WriteString("  name: '${stackName}-logs'\n")
	builder.WriteString("  location: location\n")
	builder.WriteString("  tags: tags\n")
	builder.WriteString("  properties: {\n")
	builder.WriteString("    sku: {\n")
	builder.WriteString("      name: 'PerGB2018'\n")
	builder.WriteString("    }\n")
	builder.WriteString("    retentionInDays: 30\n")
	builder.WriteString("  }\n")
	builder.WriteString("}\n\n")
	builder.WriteString("resource containerEnvironment 'Microsoft.App/managedEnvironments@2023-05-01' = {\n")
	builder.WriteString("  name: '${stackName}-env'\n")
	builder.WriteString("  location: location\n")
	builder.WriteString("  tags: tags\n")
	builder.WriteString("  properties: {\n")
	builder.WriteString("    appLogsConfiguration: {\n")
	builder.WriteString("      destination: 'log-analytics'\n")
	builder.WriteString("      logAnalyticsConfiguration: {\n")
	builder.WriteString("        customerId: workspace.properties.customerId\n")
	builder.WriteString("        sharedKey: listKeys(workspace.id, workspace.apiVersion).primarySharedKey\n")
	builder.WriteString("      }\n")
	builder.WriteString("    }\n")
	builder.WriteString("  }\n")
	builder.WriteString("}\n\n")
	builder.WriteString("resource appIdentity 'Microsoft.ManagedIdentity/userAssignedIdentities@2023-01-31' = {\n")
	builder.WriteString("  name: '${stackName}-identity'\n")
	builder.WriteString("  location: location\n")
	builder.WriteString("  tags: tags\n")
	builder.WriteString("}\n\n")
	if plan.AppConfig.KeyVaultName != "" {
		builder.WriteString("resource keyVault 'Microsoft.KeyVault/vaults@2023-02-01' = {\n")
		builder.WriteString(fmt.Sprintf("  name: '%s-%s'\n", stack, slugify(plan.AppConfig.KeyVaultName)))
		builder.WriteString("  location: location\n")
		builder.WriteString("  tags: tags\n")
		builder.WriteString("  properties: {\n")
		builder.WriteString("    tenantId: subscription().tenantId\n")
		builder.WriteString("    sku: {\n")
		builder.WriteString("      family: 'A'\n")
		builder.WriteString("      name: 'standard'\n")
		builder.WriteString("    }\n")
		builder.WriteString("    enableRbacAuthorization: false\n")
		builder.WriteString("    accessPolicies: [\n")
		builder.WriteString("      {\n")
		builder.WriteString("        tenantId: subscription().tenantId\n")
		builder.WriteString("        objectId: appIdentity.properties.principalId\n")
		builder.WriteString("        permissions: {\n")
		builder.WriteString("          secrets: [\n")
		builder.WriteString("            'Get'\n")
		builder.WriteString("            'List'\n")
		builder.WriteString("            'Set'\n")
		builder.WriteString("          ]\n")
		builder.WriteString("        }\n")
		builder.WriteString("      }\n")
		builder.WriteString("    ]\n")
		builder.WriteString("  }\n")
		builder.WriteString("}\n\n")
	}
	if plan.AppConfig.StoreName != "" {
		builder.WriteString("resource appConfigStore 'Microsoft.AppConfiguration/configurationStores@2024-05-01' = {\n")
		builder.WriteString(fmt.Sprintf("  name: '%s-%s'\n", stack, slugify(plan.AppConfig.StoreName)))
		builder.WriteString("  location: location\n")
		builder.WriteString("  tags: tags\n")
		builder.WriteString("  sku: {\n")
		builder.WriteString("    name: 'Standard'\n")
		builder.WriteString("  }\n")
		builder.WriteString("  properties: {\n")
		builder.WriteString("    publicNetworkAccess: 'Enabled'\n")
		builder.WriteString("  }\n")
		builder.WriteString("}\n\n")
		for _, env := range plan.AppConfig.Environment {
			builder.WriteString(fmt.Sprintf("resource %sKeyValue 'Microsoft.AppConfiguration/configurationStores/keyValues@2024-05-01' = {\n", resourceName(env.Name)))
			builder.WriteString(fmt.Sprintf("  name: '${appConfigStore.name}/%s'\n", env.Name))
			builder.WriteString("  properties: {\n")
			if env.SecretRef != "" && plan.AppConfig.KeyVaultName != "" {
				builder.WriteString("    contentType: 'application/vnd.microsoft.appconfig.keyvaultref+json;charset=utf-8'\n")
				builder.WriteString(fmt.Sprintf("    value: '{\"uri\":\"${keyVault.properties.vaultUri}secrets/%s\"}'\n", env.SecretRef))
			} else {
				builder.WriteString(fmt.Sprintf("    value: '%s'\n", env.Literal))
			}
			builder.WriteString("  }\n")
			builder.WriteString("}\n\n")
		}
	}
	builder.WriteString("resource containerApp 'Microsoft.App/containerApps@2023-05-01' = {\n")
	builder.WriteString("  name: containerAppName\n")
	builder.WriteString("  location: location\n")
	builder.WriteString("  tags: union(tags, {\n")
	builder.WriteString(fmt.Sprintf("    runtime: '%s'\n", plan.Compute.Runtime))
	builder.WriteString("    platform: 'container-apps'\n")
	builder.WriteString("  })\n")
	builder.WriteString("  identity: {\n")
	builder.WriteString("    type: 'UserAssigned'\n")
	builder.WriteString("    userAssignedIdentities: {\n")
	builder.WriteString("      '${appIdentity.id}': {}\n")
	builder.WriteString("    }\n")
	builder.WriteString("  }\n")
	builder.WriteString("  properties: {\n")
	builder.WriteString("    managedEnvironmentId: containerEnvironment.id\n")
	builder.WriteString("    configuration: {\n")
	if len(plan.AppConfig.Secrets) > 0 {
		builder.WriteString("      secrets: [\n")
		for _, secret := range plan.AppConfig.Secrets {
			builder.WriteString("        {\n")
			builder.WriteString(fmt.Sprintf("          name: '%s'\n", secret.Name))
			builder.WriteString(fmt.Sprintf("          keyVaultUrl: '${keyVault.properties.vaultUri}secrets/%s'\n", secret.Name))
			builder.WriteString("          identity: appIdentity.id\n")
			builder.WriteString("        }\n")
		}
		builder.WriteString("      ]\n")
	}
	builder.WriteString("      ingress: {\n")
	builder.WriteString(fmt.Sprintf("        external: %t\n", plan.Network.InternetIngress))
	builder.WriteString("        allowInsecure: false\n")
	builder.WriteString("        targetPort: containerPort\n")
	builder.WriteString("        transport: 'auto'\n")
	builder.WriteString("      }\n")
	builder.WriteString("    }\n")
	builder.WriteString("    template: {\n")
	builder.WriteString("      containers: [\n")
	builder.WriteString("        {\n")
	builder.WriteString("          name: serviceName\n")
	builder.WriteString("          image: containerImage\n")
	if len(plan.AppConfig.Environment) > 0 {
		builder.WriteString("          env: [\n")
		for _, env := range plan.AppConfig.Environment {
			builder.WriteString("            {\n")
			builder.WriteString(fmt.Sprintf("              name: '%s'\n", env.Name))
			if env.SecretRef != "" {
				builder.WriteString(fmt.Sprintf("              secretRef: '%s'\n", env.SecretRef))
			} else {
				builder.WriteString(fmt.Sprintf("              value: '%s'\n", env.Literal))
			}
			builder.WriteString("            }\n")
		}
		builder.WriteString("          ]\n")
	}
	builder.WriteString("          resources: {\n")
	builder.WriteString("            cpu: 0.5\n")
	builder.WriteString("            memory: '1Gi'\n")
	builder.WriteString("          }\n")
	builder.WriteString("          probes: [\n")
	builder.WriteString("            {\n")
	builder.WriteString("              type: 'Liveness'\n")
	builder.WriteString("              httpGet: {\n")
	builder.WriteString(fmt.Sprintf("                path: '%s'\n", plan.Compute.HealthCheckPath))
	builder.WriteString("                port: containerPort\n")
	builder.WriteString("              }\n")
	builder.WriteString("              initialDelaySeconds: 15\n")
	builder.WriteString("              periodSeconds: 30\n")
	builder.WriteString("            }\n")
	builder.WriteString("            {\n")
	builder.WriteString("              type: 'Readiness'\n")
	builder.WriteString("              httpGet: {\n")
	builder.WriteString(fmt.Sprintf("                path: '%s'\n", plan.Compute.HealthCheckPath))
	builder.WriteString("                port: containerPort\n")
	builder.WriteString("              }\n")
	builder.WriteString("              initialDelaySeconds: 10\n")
	builder.WriteString("              periodSeconds: 15\n")
	builder.WriteString("            }\n")
	builder.WriteString("          ]\n")
	builder.WriteString("        }\n")
	builder.WriteString("      ]\n")
	builder.WriteString("      scale: {\n")
	builder.WriteString("        minReplicas: minReplicas\n")
	builder.WriteString("        maxReplicas: maxReplicas\n")
	builder.WriteString("      }\n")
	builder.WriteString("    }\n")
	builder.WriteString("  }\n")
	builder.WriteString("}\n")

	for _, resource := range plan.Resources {
		switch resource.Kind {
		case "database":
			if !resource.Managed {
				continue
			}
			builder.WriteString(fmt.Sprintf("\nresource %sPostgres 'Microsoft.DBforPostgreSQL/flexibleServers@2023-06-01-preview' = {\n  name: '%s-%s'\n  location: location\n  sku: {\n    name: '%s'\n    tier: 'Burstable'\n  }\n  properties: {\n    version: '15'\n    administratorLogin: 'appadmin'\n    administratorLoginPassword: databaseAdminPassword\n  }\n}\n", resource.Name, stack, resource.Name, azurePostgresSKU(resource.Class)))
		case "cache":
			if !resource.Managed {
				continue
			}
			builder.WriteString(fmt.Sprintf("\nresource %sRedis 'Microsoft.Cache/Redis@2023-08-01' = {\n  name: '%s-%s'\n  location: location\n  properties: {\n    sku: {\n      name: 'Basic'\n      family: 'C'\n      capacity: %d\n    }\n    enableNonSslPort: false\n  }\n}\n", resource.Name, stack, resource.Name, azureRedisCapacity(resource.Class)))
		case "queue":
			if !resource.Managed {
				continue
			}
			builder.WriteString(fmt.Sprintf("\nresource %sServiceBus 'Microsoft.ServiceBus/namespaces@2022-10-01-preview' = {\n  name: '%s-%s'\n  location: location\n  sku: {\n    name: 'Basic'\n    tier: 'Basic'\n  }\n}\n\nresource %sQueue 'Microsoft.ServiceBus/namespaces/queues@2022-10-01-preview' = {\n  name: '${%sServiceBus.name}/%s'\n  properties: {}\n}\n", resource.Name, stack, resource.Name, resource.Name, resource.Name, resource.Name))
		case "key_vault":
			continue
		default:
			definition, ok := lookupAzureService(resource.Kind)
			if !ok {
				builder.WriteString(fmt.Sprintf("\n// unsupported azure resource kind: %s\n", resource.Kind))
				continue
			}
			builder.WriteString(renderGenericAzureResource(definition, resource, stack))
		}
	}
	builder.WriteString("\noutput serviceName string = containerApp.name\n")
	builder.WriteString("output serviceUrl string = 'https://${containerApp.properties.configuration.ingress.fqdn}'\n")
	builder.WriteString("output managedEnvironmentId string = containerEnvironment.id\n")
	if plan.AppConfig.StoreName != "" {
		builder.WriteString("output appConfigurationEndpoint string = appConfigStore.properties.endpoint\n")
	}
	return builder.String(), nil
}

func azureRedisCapacity(class string) int {
	switch class {
	case "medium":
		return 1
	default:
		return 0
	}
}

func renderGenericAzureResource(definition azureServiceDefinition, resource ResourcePlan, stack string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("\nresource %s '%s@%s' = {\n", resource.Name, definition.ARMType, definition.APIVersion))
	builder.WriteString(fmt.Sprintf("  name: '%s-%s'\n", stack, resource.Name))
	if definition.Location {
		builder.WriteString("  location: location\n")
	}
	if definition.Body != "" {
		lines := strings.Split(definition.Body, "\n")
		for _, line := range lines {
			builder.WriteString("  ")
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	} else {
		builder.WriteString("  properties: {}\n")
	}
	builder.WriteString("}\n")
	return builder.String()
}

func resourceName(value string) string {
	out := strings.ReplaceAll(slugify(value), "-", "_")
	if out == "" {
		return "resource_ref"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "r_" + out
	}
	return out
}
