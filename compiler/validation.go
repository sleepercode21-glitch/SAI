package compiler

import (
	"fmt"
	"strings"

	"github.com/sleepercode/sai/compiler/infra"
)

func validateLowered(result *Result) error {
	if result == nil || result.Lowered == nil || result.Lowered.Infra == nil || result.Lowered.Deploy == nil {
		return fmt.Errorf("compiler validation requires lowered infra and deploy plans")
	}
	if err := validateCommon(result.Lowered.Infra, result.Lowered.Deploy.ContextDir); err != nil {
		return err
	}
	switch result.Lowered.Infra.Cloud {
	case "azure":
		return validateAzurePlan(result.Lowered.Infra)
	case "aws":
		return validateAWSPlan(result.Lowered.Infra)
	case "gcp":
		return validateGCPPlan(result.Lowered.Infra)
	default:
		return fmt.Errorf("compiler validation does not support cloud %q", result.Lowered.Infra.Cloud)
	}
}

func validateCommon(plan *infra.Plan, contextDir string) error {
	if strings.TrimSpace(plan.Region) == "" {
		return fmt.Errorf("compiler validation: region must be set")
	}
	if strings.TrimSpace(contextDir) == "" {
		return fmt.Errorf("compiler validation: service path must be set for deployable workloads")
	}
	if plan.Network.ServicePort < 1 || plan.Network.ServicePort > 65535 {
		return fmt.Errorf("compiler validation: service port %d is out of range", plan.Network.ServicePort)
	}
	if !strings.HasPrefix(plan.Compute.HealthCheckPath, "/") {
		return fmt.Errorf("compiler validation: health check path %q must start with /", plan.Compute.HealthCheckPath)
	}
	return nil
}

func validateAzurePlan(plan *infra.Plan) error {
	keyVaultCount := 0
	for _, resource := range plan.Resources {
		switch resource.Kind {
		case "database", "cache", "queue":
		case "key_vault":
			keyVaultCount++
		default:
			if _, ok := infra.LookupAzureService(resource.Kind); !ok {
				return fmt.Errorf("compiler validation: azure does not support resource kind %q", resource.Kind)
			}
		}
	}
	if keyVaultCount > 1 {
		return fmt.Errorf("compiler validation: azure supports at most one key_vault resource per service")
	}
	return nil
}

func validateAWSPlan(plan *infra.Plan) error {
	for _, resource := range plan.Resources {
		switch resource.Kind {
		case "database", "cache", "queue":
		default:
			return fmt.Errorf("compiler validation: aws backend does not support resource kind %q", resource.Kind)
		}
	}
	return nil
}

func validateGCPPlan(plan *infra.Plan) error {
	for _, resource := range plan.Resources {
		switch resource.Kind {
		case "database":
		default:
			return fmt.Errorf("compiler validation: gcp backend does not support resource kind %q", resource.Kind)
		}
	}
	return nil
}
