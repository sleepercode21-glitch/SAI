package infra

import (
	"strings"
	"testing"

	"github.com/sleepercode/sai/ir"
	"github.com/sleepercode/sai/planner"
)

func TestGenerateTerraformJSONIncludesCoreResources(t *testing.T) {
	program := &ir.ProgramIR{
		Application: ir.ApplicationIR{
			Name:      "orders",
			Cloud:     "aws",
			Region:    "us-east-1",
			BudgetUSD: 75,
			Env:       "prod",
		},
		Service: ir.ServiceIR{
			Name:            "api",
			Runtime:         "node",
			Port:            3000,
			Exposure:        ir.ExposurePublicHTTP,
			HealthCheckPath: "/health",
			Connects:        []string{"postgres"},
		},
		Resources: []ir.ResourceIR{
			{Name: "postgres", Kind: "database", Type: "managed", Size: "small"},
		},
	}
	deployment := &planner.Plan{
		Profile:      planner.ProfileBalancedWeb,
		InfraClass:   "managed-small",
		MinInstances: 1,
		MaxInstances: 3,
		EstimatedUSD: 75,
	}

	plan, err := Lower(program, deployment)
	if err != nil {
		t.Fatalf("Lower returned error: %v", err)
	}

	data, err := GenerateTerraformJSON(plan)
	if err != nil {
		t.Fatalf("GenerateTerraformJSON returned error: %v", err)
	}

	got := string(data)
	for _, fragment := range []string{
		`"aws_vpc"`,
		`"aws_ecs_service"`,
		`"aws_db_instance"`,
		`"engine": "postgres"`,
		`"container_definitions"`,
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("expected terraform JSON to contain %q", fragment)
		}
	}
}

func TestGenerateTerraformJSONRejectsUnsupportedCloud(t *testing.T) {
	_, err := GenerateTerraformJSON(&Plan{Cloud: "gcp"})
	if err == nil {
		t.Fatal("expected unsupported cloud to fail")
	}
}
