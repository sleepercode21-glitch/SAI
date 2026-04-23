package build

import (
	"fmt"
	"path/filepath"

	"github.com/sleepercode/sai/ir"
	"github.com/sleepercode/sai/planner"
)

type HealthcheckPlan struct {
	Path     string `json:"path"`
	Port     int    `json:"port"`
	Interval string `json:"interval"`
	Timeout  string `json:"timeout"`
}

// Plan is the lowering target for build generation.
type Plan struct {
	ServiceName     string          `json:"service_name"`
	Runtime         string          `json:"runtime"`
	ContextDir      string          `json:"context_dir"`
	DockerfilePath  string          `json:"dockerfile_path"`
	ImageRepository string          `json:"image_repository"`
	ImageTag        string          `json:"image_tag"`
	ExposedPort     int             `json:"exposed_port"`
	Healthcheck     HealthcheckPlan `json:"healthcheck"`
}

// Lower converts IR into a build-specific plan without leaking infra concerns.
func Lower(program *ir.ProgramIR, deployment *planner.Plan) (*Plan, error) {
	tag := fmt.Sprintf("%s-%s", program.Application.Env, deployment.Profile)
	return &Plan{
		ServiceName:     program.Service.Name,
		Runtime:         program.Service.Runtime,
		ContextDir:      program.Service.Path,
		DockerfilePath:  filepath.Join(program.Service.Path, "Dockerfile"),
		ImageRepository: fmt.Sprintf("%s/%s", program.Application.Slug, program.Service.Name),
		ImageTag:        tag,
		ExposedPort:     program.Service.Port,
		Healthcheck: HealthcheckPlan{
			Path:     program.Service.HealthCheckPath,
			Port:     program.Service.Port,
			Interval: "30s",
			Timeout:  "5s",
		},
	}, nil
}
