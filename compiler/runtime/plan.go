package runtime

import (
	"fmt"

	"github.com/sleepercode/sai/ir"
	"github.com/sleepercode/sai/planner"
)

type EnvironmentVariable struct {
	Name   string `json:"name"`
	Source string `json:"source"`
}

type HealthPlan struct {
	Path             string `json:"path"`
	Port             int    `json:"port"`
	InitialDelaySecs int    `json:"initial_delay_secs"`
	IntervalSecs     int    `json:"interval_secs"`
	FailureThreshold int    `json:"failure_threshold"`
}

// Plan is the lowering target for runtime generation.
type Plan struct {
	Platform     string                `json:"platform"`
	ServiceName  string                `json:"service_name"`
	Port         int                   `json:"port"`
	Exposure     ir.ExposureKind       `json:"exposure"`
	MinInstances int                   `json:"min_instances"`
	MaxInstances int                   `json:"max_instances"`
	Health       HealthPlan            `json:"health"`
	Environment  []EnvironmentVariable `json:"environment"`
}

// Lower converts IR into a runtime-specific plan.
func Lower(program *ir.ProgramIR, deployment *planner.Plan) (*Plan, error) {
	env := []EnvironmentVariable{
		{Name: "PORT", Source: fmt.Sprintf("literal:%d", program.Service.Port)},
		{Name: "SAI_ENV", Source: fmt.Sprintf("literal:%s", program.Application.Env)},
	}

	for _, resource := range program.Resources {
		if !connectsTo(program.Service.Connects, resource.Name) {
			continue
		}
		env = append(env, environmentVariablesForResource(resource)...)
	}

	return &Plan{
		Platform:     "managed-container",
		ServiceName:  program.Service.Name,
		Port:         program.Service.Port,
		Exposure:     program.Service.Exposure,
		MinInstances: deployment.MinInstances,
		MaxInstances: deployment.MaxInstances,
		Health: HealthPlan{
			Path:             program.Service.HealthCheckPath,
			Port:             program.Service.Port,
			InitialDelaySecs: 5,
			IntervalSecs:     30,
			FailureThreshold: 3,
		},
		Environment: env,
	}, nil
}

func environmentVariablesForResource(resource ir.ResourceIR) []EnvironmentVariable {
	prefix := upperSnake(resource.Name)
	switch resource.Kind {
	case "database":
		return []EnvironmentVariable{{
			Name:   fmt.Sprintf("%s_URL", prefix),
			Source: fmt.Sprintf("secret:%s.connection_string", resource.Name),
		}}
	case "cache":
		return []EnvironmentVariable{
			{Name: fmt.Sprintf("%s_ENDPOINT", prefix), Source: fmt.Sprintf("secret:%s.endpoint", resource.Name)},
			{Name: fmt.Sprintf("%s_PASSWORD", prefix), Source: fmt.Sprintf("secret:%s.password", resource.Name)},
		}
	case "queue":
		return []EnvironmentVariable{
			{Name: fmt.Sprintf("%s_NAMESPACE", prefix), Source: fmt.Sprintf("secret:%s.namespace", resource.Name)},
			{Name: fmt.Sprintf("%s_NAME", prefix), Source: fmt.Sprintf("literal:%s", resource.Name)},
		}
	default:
		return []EnvironmentVariable{{
			Name:   fmt.Sprintf("%s_REF", prefix),
			Source: fmt.Sprintf("secret:%s.reference", resource.Name),
		}}
	}
}

func connectsTo(resources []string, target string) bool {
	for _, resource := range resources {
		if resource == target {
			return true
		}
	}
	return false
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
