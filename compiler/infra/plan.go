package infra

import (
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

// Plan is the lowering target for infrastructure generation.
type Plan struct {
	ApplicationName string        `json:"application_name"`
	Environment     string        `json:"environment"`
	Cloud           string        `json:"cloud"`
	Region          string        `json:"region"`
	EstimatedUSD    int           `json:"estimated_monthly_usd"`
	Network         NetworkPlan   `json:"network"`
	Compute         ComputePlan   `json:"compute"`
	Database        *DatabasePlan `json:"database,omitempty"`
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
	}

	for _, resource := range program.Resources {
		if resource.Kind != "database" {
			continue
		}
		plan.Database = &DatabasePlan{
			Name:      resource.Name,
			Engine:    resource.Kind,
			Class:     resource.Size,
			Managed:   resource.Type == "managed",
			Connected: connectsTo(program.Service.Connects, resource.Name),
		}
		break
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
