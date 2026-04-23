package deploy

import (
	"fmt"

	"github.com/sleepercode/sai/ir"
	"github.com/sleepercode/sai/planner"
)

// Plan is the lowering target for deployment generation.
type Plan struct {
	Strategy          string   `json:"strategy"`
	Environment       string   `json:"environment"`
	SerializedKey     string   `json:"serialized_key"`
	SupportsLocalCLI  bool     `json:"supports_local_cli"`
	SupportsGitHub    bool     `json:"supports_github"`
	SupportsBitbucket bool     `json:"supports_bitbucket"`
	BuildArtifact     string   `json:"build_artifact"`
	PreDeployChecks   []string `json:"pre_deploy_checks"`
	PostDeployChecks  []string `json:"post_deploy_checks"`
}

// Lower converts IR into a deploy-specific plan.
func Lower(program *ir.ProgramIR, deployment *planner.Plan) (*Plan, error) {
	return &Plan{
		Strategy:          fmt.Sprintf("rolling-%s", deployment.Profile),
		Environment:       program.Application.Env,
		SerializedKey:     fmt.Sprintf("%s/%s", program.Application.Slug, program.Application.Env),
		SupportsLocalCLI:  true,
		SupportsGitHub:    true,
		SupportsBitbucket: true,
		BuildArtifact:     fmt.Sprintf("%s:%s-%s", program.Application.Slug+"/"+program.Service.Name, program.Application.Env, deployment.Profile),
		PreDeployChecks: []string{
			"validate-manifest",
			"build-container-image",
			"terraform-plan",
		},
		PostDeployChecks: []string{
			"http-health-check",
		},
	}, nil
}
