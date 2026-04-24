package runtime

import (
	"testing"

	"github.com/sleepercode/sai/ir"
	"github.com/sleepercode/sai/planner"
)

func TestLowerAddsEnvironmentBindingsForDatabaseCacheAndQueue(t *testing.T) {
	program := &ir.ProgramIR{
		Application: ir.ApplicationIR{
			Env: "prod",
		},
		Service: ir.ServiceIR{
			Name:            "api",
			Port:            3000,
			Exposure:        ir.ExposurePublicHTTP,
			HealthCheckPath: "/health",
			Connects:        []string{"postgres", "cache", "jobs"},
		},
		Resources: []ir.ResourceIR{
			{Name: "postgres", Kind: "database"},
			{Name: "cache", Kind: "cache"},
			{Name: "jobs", Kind: "queue"},
		},
	}
	deployment := &planner.Plan{
		MinInstances: 1,
		MaxInstances: 3,
	}

	plan, err := Lower(program, deployment)
	if err != nil {
		t.Fatalf("Lower returned error: %v", err)
	}

	assertEnvPresent(t, plan.Environment, "POSTGRES_URL", "secret:postgres.connection_string")
	assertEnvPresent(t, plan.Environment, "CACHE_ENDPOINT", "secret:cache.endpoint")
	assertEnvPresent(t, plan.Environment, "CACHE_PASSWORD", "secret:cache.password")
	assertEnvPresent(t, plan.Environment, "JOBS_NAMESPACE", "secret:jobs.namespace")
	assertEnvPresent(t, plan.Environment, "JOBS_NAME", "literal:jobs")
}

func assertEnvPresent(t *testing.T, env []EnvironmentVariable, name, source string) {
	t.Helper()
	for _, item := range env {
		if item.Name == name && item.Source == source {
			return
		}
	}
	t.Fatalf("expected environment variable %s=%s", name, source)
}
