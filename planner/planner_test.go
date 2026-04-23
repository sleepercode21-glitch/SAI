package planner

import (
	"testing"

	"github.com/sleepercode/sai/ir"
)

func TestBuildChoosesExpectedProfiles(t *testing.T) {
	testCases := []struct {
		name      string
		users     int
		budgetUSD int
		want      DeploymentProfile
		wantClass string
	}{
		{name: "dev-min", users: 1000, budgetUSD: 25, want: ProfileDevMin, wantClass: "shared-small"},
		{name: "balanced-web", users: 5000, budgetUSD: 75, want: ProfileBalancedWeb, wantClass: "managed-small"},
		{name: "bursty-web", users: 25000, budgetUSD: 180, want: ProfileBurstyWeb, wantClass: "managed-medium"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := Build(&ir.ProgramIR{
				Application: ir.ApplicationIR{
					Users:     tc.users,
					BudgetUSD: tc.budgetUSD,
				},
			})
			if err != nil {
				t.Fatalf("Build returned error: %v", err)
			}
			if got, want := plan.Profile, tc.want; got != want {
				t.Fatalf("unexpected profile: got %q want %q", got, want)
			}
			if got, want := plan.InfraClass, tc.wantClass; got != want {
				t.Fatalf("unexpected infra class: got %q want %q", got, want)
			}
		})
	}
}

func TestBuildRejectsIncompatibleConstraints(t *testing.T) {
	_, err := Build(&ir.ProgramIR{
		Application: ir.ApplicationIR{
			Users:     50000,
			BudgetUSD: 50,
		},
	})
	if err == nil {
		t.Fatal("expected Build to fail for incompatible constraints")
	}
}
