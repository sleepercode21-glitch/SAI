package infra

import (
	"strconv"

	"github.com/sleepercode/sai/ir"
	"github.com/sleepercode/sai/planner"
)

type NetworkPlan struct {
	InternetIngress bool `json:"internet_ingress"`
	ServicePort     int  `json:"service_port"`
}

type ComputePlan struct {
	Platform        string `json:"platform"`
	ServiceName     string `json:"service_name"`
	Runtime         string `json:"runtime"`
	InfraClass      string `json:"infra_class"`
	MinInstances    int    `json:"min_instances"`
	MaxInstances    int    `json:"max_instances"`
	HealthCheckPath string `json:"health_check_path"`
}

type DatabasePlan struct {
	Name      string `json:"name"`
	Engine    string `json:"engine"`
	Class     string `json:"class"`
	Managed   bool   `json:"managed"`
	Connected bool   `json:"connected"`
}

type ResourcePlan struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Engine    string `json:"engine,omitempty"`
	Class     string `json:"class"`
	Managed   bool   `json:"managed"`
	Connected bool   `json:"connected"`
}

type AppSecretPlan struct {
	Name        string `json:"name"`
	Placeholder string `json:"placeholder"`
}

type AppEnvironmentPlan struct {
	Name      string `json:"name"`
	Literal   string `json:"literal,omitempty"`
	SecretRef string `json:"secret_ref,omitempty"`
}

type AppConfigPlan struct {
	KeyVaultName string               `json:"key_vault_name,omitempty"`
	StoreName    string               `json:"store_name,omitempty"`
	Secrets      []AppSecretPlan      `json:"secrets,omitempty"`
	Environment  []AppEnvironmentPlan `json:"environment,omitempty"`
}

// Plan is the lowering target for infrastructure generation.
type Plan struct {
	ApplicationName string         `json:"application_name"`
	Environment     string         `json:"environment"`
	Cloud           string         `json:"cloud"`
	Region          string         `json:"region"`
	EstimatedUSD    int            `json:"estimated_monthly_usd"`
	Network         NetworkPlan    `json:"network"`
	Compute         ComputePlan    `json:"compute"`
	Database        *DatabasePlan  `json:"database,omitempty"`
	AppConfig       AppConfigPlan  `json:"app_config"`
	Resources       []ResourcePlan `json:"resources,omitempty"`
}

// Lower converts the typed IR and planner output into an infra-specific plan.
func Lower(program *ir.ProgramIR, deployment *planner.Plan) (*Plan, error) {
	plan := &Plan{
		ApplicationName: program.Application.Name,
		Environment:     program.Application.Env,
		Cloud:           program.Application.Cloud,
		Region:          program.Application.Region,
		EstimatedUSD:    deployment.EstimatedUSD,
		Network: NetworkPlan{
			InternetIngress: program.Service.Exposure == ir.ExposurePublicHTTP,
			ServicePort:     program.Service.Port,
		},
		Compute: ComputePlan{
			Platform:        "managed-container",
			ServiceName:     program.Service.Name,
			Runtime:         program.Service.Runtime,
			InfraClass:      deployment.InfraClass,
			MinInstances:    deployment.MinInstances,
			MaxInstances:    deployment.MaxInstances,
			HealthCheckPath: program.Service.HealthCheckPath,
		},
		AppConfig: AppConfigPlan{
			Environment: []AppEnvironmentPlan{
				{Name: "PORT", Literal: intString(program.Service.Port)},
				{Name: "SAI_ENV", Literal: program.Application.Env},
			},
		},
	}

	for _, resource := range program.Resources {
		resourcePlan := ResourcePlan{
			Name:      resource.Name,
			Kind:      resource.Kind,
			Class:     resource.Size,
			Managed:   resource.Type == "managed",
			Connected: connectsTo(program.Service.Connects, resource.Name),
		}
		if resource.Kind == "database" {
			resourcePlan.Engine = inferDatabaseEngine(resource)
			plan.Database = &DatabasePlan{
				Name:      resourcePlan.Name,
				Engine:    resourcePlan.Engine,
				Class:     resourcePlan.Class,
				Managed:   resourcePlan.Managed,
				Connected: resourcePlan.Connected,
			}
		}
		if resource.Kind == "key_vault" {
			plan.AppConfig.KeyVaultName = resource.Name
		}
		if resourcePlan.Connected {
			secrets, env := appConfigForResource(resourcePlan)
			plan.AppConfig.Secrets = append(plan.AppConfig.Secrets, secrets...)
			plan.AppConfig.Environment = append(plan.AppConfig.Environment, env...)
		}
		plan.Resources = append(plan.Resources, resourcePlan)
	}

	if plan.Cloud == "azure" && len(plan.AppConfig.Secrets) > 0 && plan.AppConfig.KeyVaultName == "" {
		plan.AppConfig.KeyVaultName = "app"
	}
	if plan.Cloud == "azure" && len(plan.AppConfig.Environment) > 0 {
		plan.AppConfig.StoreName = "appconfig"
	}

	return plan, nil
}

func connectsTo(resources []string, target string) bool {
	for _, resource := range resources {
		if resource == target {
			return true
		}
	}
	return false
}

func inferDatabaseEngine(resource ir.ResourceIR) string {
	if resource.Kind != "database" {
		return resource.Kind
	}
	return "postgres"
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func appConfigForResource(resource ResourcePlan) ([]AppSecretPlan, []AppEnvironmentPlan) {
	prefix := upperSnake(resource.Name)
	switch resource.Kind {
	case "database":
		return []AppSecretPlan{
				{Name: secretName(resource.Name, "connection-string"), Placeholder: "postgres://app:change-me@" + resource.Name + ":5432/app"},
			}, []AppEnvironmentPlan{
				{Name: prefix + "_URL", SecretRef: secretName(resource.Name, "connection-string")},
			}
	case "cache":
		return []AppSecretPlan{
				{Name: secretName(resource.Name, "endpoint"), Placeholder: resource.Name + ".cache.internal:6379"},
				{Name: secretName(resource.Name, "password"), Placeholder: "change-me-cache-password"},
			}, []AppEnvironmentPlan{
				{Name: prefix + "_ENDPOINT", SecretRef: secretName(resource.Name, "endpoint")},
				{Name: prefix + "_PASSWORD", SecretRef: secretName(resource.Name, "password")},
			}
	case "queue":
		return []AppSecretPlan{
				{Name: secretName(resource.Name, "namespace"), Placeholder: resource.Name + ".queue.internal"},
			}, []AppEnvironmentPlan{
				{Name: prefix + "_NAMESPACE", SecretRef: secretName(resource.Name, "namespace")},
				{Name: prefix + "_NAME", Literal: resource.Name},
			}
	default:
		return []AppSecretPlan{
				{Name: secretName(resource.Name, "reference"), Placeholder: "https://example.invalid/" + resource.Name},
			}, []AppEnvironmentPlan{
				{Name: prefix + "_REF", SecretRef: secretName(resource.Name, "reference")},
			}
	}
}

func secretName(resource string, suffix string) string {
	return slugify(resource) + "-" + suffix
}

func upperSnake(value string) string {
	out := make([]rune, 0, len(value))
	for _, ch := range value {
		switch {
		case ch >= 'a' && ch <= 'z':
			out = append(out, ch-'a'+'A')
		case ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9':
			out = append(out, ch)
		default:
			out = append(out, '_')
		}
	}
	return string(out)
}
