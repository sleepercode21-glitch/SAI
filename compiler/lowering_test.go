package compiler

import "testing"

func TestLowerBuildsIndependentPlans(t *testing.T) {
	result, err := PlanFile("../examples/orders.sai")
	if err != nil {
		t.Fatalf("PlanFile returned error: %v", err)
	}

	if result.Lowered == nil {
		t.Fatal("expected lowered plans to be present")
	}
	if result.Lowered.Infra == nil || result.Lowered.Build == nil || result.Lowered.Runtime == nil || result.Lowered.Deploy == nil || result.Lowered.Incident == nil {
		t.Fatal("expected all lowered plans to be populated")
	}
	if got, want := result.Lowered.Infra.Compute.InfraClass, "managed-small"; got != want {
		t.Fatalf("unexpected infra class: got %q want %q", got, want)
	}
	if got, want := result.Lowered.Runtime.Port, 3000; got != want {
		t.Fatalf("unexpected runtime port: got %d want %d", got, want)
	}
	if got, want := result.Lowered.Build.Healthcheck.Path, "/health"; got != want {
		t.Fatalf("unexpected build healthcheck path: got %q want %q", got, want)
	}
	if got, want := result.Lowered.Incident.Responses[0].Action, "restart"; got != want {
		t.Fatalf("unexpected incident action: got %q want %q", got, want)
	}
}
