package compiler

import (
	buildplan "github.com/sleepercode/sai/compiler/build"
	deployplan "github.com/sleepercode/sai/compiler/deploy"
	incidentplan "github.com/sleepercode/sai/compiler/incident"
	infraplan "github.com/sleepercode/sai/compiler/infra"
	runtimeplan "github.com/sleepercode/sai/compiler/runtime"
	"github.com/sleepercode/sai/ir"
	"github.com/sleepercode/sai/planner"
)

// LoweredPlans separates all downstream targets so code generators never read AST directly.
type LoweredPlans struct {
	Infra    *infraplan.Plan    `json:"infra"`
	Build    *buildplan.Plan    `json:"build"`
	Runtime  *runtimeplan.Plan  `json:"runtime"`
	Deploy   *deployplan.Plan   `json:"deploy"`
	Incident *incidentplan.Plan `json:"incident"`
}

// Lower converts IR plus planner output into independent downstream plans.
func Lower(program *ir.ProgramIR, deployment *planner.Plan) (*LoweredPlans, error) {
	infraPlan, err := infraplan.Lower(program, deployment)
	if err != nil {
		return nil, err
	}

	buildPlan, err := buildplan.Lower(program, deployment)
	if err != nil {
		return nil, err
	}

	runtimePlan, err := runtimeplan.Lower(program, deployment)
	if err != nil {
		return nil, err
	}

	deployPlan, err := deployplan.Lower(program, deployment)
	if err != nil {
		return nil, err
	}

	incidentPlan, err := incidentplan.Lower(program, deployment)
	if err != nil {
		return nil, err
	}

	return &LoweredPlans{
		Infra:    infraPlan,
		Build:    buildPlan,
		Runtime:  runtimePlan,
		Deploy:   deployPlan,
		Incident: incidentPlan,
	}, nil
}
