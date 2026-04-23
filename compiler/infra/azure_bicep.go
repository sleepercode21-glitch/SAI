package infra

import (
	"fmt"
	"strings"
)

func generateAzureBicep(plan *Plan) (string, error) {
	stack := slugify(fmt.Sprintf("%s-%s", plan.ApplicationName, plan.Environment))
	var builder strings.Builder
	builder.WriteString("targetScope = 'resourceGroup'\n\n")
	builder.WriteString(fmt.Sprintf("param location string = '%s'\n\n", plan.Region))
	builder.WriteString(fmt.Sprintf("resource containerEnvironment 'Microsoft.App/managedEnvironments@2023-05-01' = {\n  name: '%s-env'\n  location: location\n  properties: {}\n}\n\n", stack))
	builder.WriteString(fmt.Sprintf("resource containerApp 'Microsoft.App/containerApps@2023-05-01' = {\n  name: '%s-%s'\n  location: location\n  properties: {\n    managedEnvironmentId: containerEnvironment.id\n    configuration: {\n      ingress: {\n        external: %t\n        targetPort: %d\n      }\n    }\n    template: {\n      containers: [\n        {\n          name: '%s'\n          image: '%s/%s:latest'\n          probes: [\n            {\n              type: 'Liveness'\n              httpGet: {\n                path: '%s'\n                port: %d\n              }\n            }\n          ]\n        }\n      ]\n      scale: {\n        minReplicas: %d\n        maxReplicas: %d\n      }\n    }\n  }\n}\n",
		stack, plan.Compute.ServiceName, plan.Network.InternetIngress, plan.Network.ServicePort, plan.Compute.ServiceName, slugify(plan.ApplicationName), plan.Compute.ServiceName, plan.Compute.HealthCheckPath, plan.Network.ServicePort, plan.Compute.MinInstances, plan.Compute.MaxInstances))
	if plan.Database != nil && plan.Database.Managed {
		builder.WriteString(fmt.Sprintf("\nresource postgresServer 'Microsoft.DBforPostgreSQL/flexibleServers@2023-06-01-preview' = {\n  name: '%s-%s'\n  location: location\n  sku: {\n    name: '%s'\n    tier: 'Burstable'\n  }\n  properties: {\n    version: '15'\n    administratorLogin: 'appadmin'\n    administratorLoginPassword: 'change-me-in-keyvault'\n  }\n}\n", stack, plan.Database.Name, azurePostgresSKU(plan.Database.Class)))
	}
	builder.WriteString("\noutput serviceName string = containerApp.name\n")
	return builder.String(), nil
}
