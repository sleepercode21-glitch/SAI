package planner

import (
	"fmt"

	"github.com/sleepercode/sai/ir"
)

type DeploymentProfile string

const (
	ProfileDevMin      DeploymentProfile = "dev-min"
	ProfileBalancedWeb DeploymentProfile = "balanced-web"
	ProfileBurstyWeb   DeploymentProfile = "bursty-web"
)

type Plan struct {
	Profile      DeploymentProfile `json:"profile"`
	InfraClass   string            `json:"infra_class"`
	MinInstances int               `json:"min_instances"`
	MaxInstances int               `json:"max_instances"`
	EstimatedUSD int               `json:"estimated_monthly_usd"`
}

// Build chooses a bounded deployment profile from deterministic rules.
func Build(program *ir.ProgramIR) (*Plan, error) {
	users := program.Application.Users
	budget := program.Application.BudgetUSD

	switch {
	case users <= 1000 && budget >= 25:
		return &Plan{
			Profile:      ProfileDevMin,
			InfraClass:   "shared-small",
			MinInstances: 1,
			MaxInstances: 1,
			EstimatedUSD: 25,
		}, nil
	case users <= 10000 && budget >= 75:
		return &Plan{
			Profile:      ProfileBalancedWeb,
			InfraClass:   "managed-small",
			MinInstances: 1,
			MaxInstances: 3,
			EstimatedUSD: 75,
		}, nil
	case users <= 50000 && budget >= 180:
		return &Plan{
			Profile:      ProfileBurstyWeb,
			InfraClass:   "managed-medium",
			MinInstances: 2,
			MaxInstances: 6,
			EstimatedUSD: 180,
		}, nil
	default:
		return nil, fmt.Errorf("planner could not find a compatible deployment profile for %d users and $%d/month", users, budget)
	}
}
