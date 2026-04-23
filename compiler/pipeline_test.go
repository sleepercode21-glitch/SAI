package compiler

import "testing"

func TestCompileFilePipeline(t *testing.T) {
	result, err := CompileFile("../examples/orders.sai")
	if err != nil {
		t.Fatalf("CompileFile returned error: %v", err)
	}

	if got, want := result.IR.Application.Name, "orders"; got != want {
		t.Fatalf("unexpected application name: got %q want %q", got, want)
	}
	if got, want := result.IR.Service.Runtime, "node"; got != want {
		t.Fatalf("unexpected runtime: got %q want %q", got, want)
	}
	if got, want := result.IR.Service.Exposure, "public_http"; string(got) != want {
		t.Fatalf("unexpected exposure: got %q want %q", got, want)
	}
}

func TestPlanFileChoosesExpectedProfile(t *testing.T) {
	result, err := PlanFile("../examples/orders.sai")
	if err != nil {
		t.Fatalf("PlanFile returned error: %v", err)
	}

	if got, want := string(result.Plan.Profile), "balanced-web"; got != want {
		t.Fatalf("unexpected profile: got %q want %q", got, want)
	}
	if result.InfraArtifact == nil {
		t.Fatal("expected provider-native infra artifact to be present")
	}
	if got, want := string(result.InfraArtifact.Format), "terraform-json"; got != want {
		t.Fatalf("unexpected infra artifact format: got %q want %q", got, want)
	}
}
