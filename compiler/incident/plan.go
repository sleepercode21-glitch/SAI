package incident

import (
	"github.com/sleepercode/sai/ir"
	"github.com/sleepercode/sai/planner"
)

type TriggerPlan struct {
	Name      string `json:"name"`
	Condition string `json:"condition"`
}

type ResponsePlan struct {
	Action      string `json:"action"`
	MaxAttempts int    `json:"max_attempts"`
}

// Plan is the lowering target for incident generation.
type Plan struct {
	ServiceName string         `json:"service_name"`
	Triggers    []TriggerPlan  `json:"triggers"`
	Responses   []ResponsePlan `json:"responses"`
}

// Lower converts IR into a deterministic incident-response plan.
func Lower(program *ir.ProgramIR, _ *planner.Plan) (*Plan, error) {
	return &Plan{
		ServiceName: program.Service.Name,
		Triggers: []TriggerPlan{
			{Name: "failed-health-check", Condition: "health endpoint returns non-success"},
			{Name: "crash-loop", Condition: "container restarts exceed threshold"},
		},
		Responses: []ResponsePlan{
			{Action: "restart", MaxAttempts: 1},
			{Action: "rollback", MaxAttempts: 1},
		},
	}, nil
}
