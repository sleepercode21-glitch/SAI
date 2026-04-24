package executor

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	deployartifact "github.com/sleepercode/sai/compiler/deploy"
)

type fakeCommander struct {
	commands []CommandSpec
	err      error
}

func (f *fakeCommander) Run(_ context.Context, spec CommandSpec) error {
	f.commands = append(f.commands, spec)
	return f.err
}

func (f *fakeCommander) RunWithWriters(_ context.Context, spec CommandSpec, _ io.Writer, _ io.Writer) error {
	f.commands = append(f.commands, spec)
	return f.err
}

func TestExecuteBundleRunsAzureDeployScript(t *testing.T) {
	commander := &fakeCommander{}
	exec := &Executor{Commander: commander}

	result, err := exec.ExecuteBundle(context.Background(), "/tmp/out", &deployartifact.Bundle{Provider: "azure"})
	if err != nil {
		t.Fatalf("ExecuteBundle returned error: %v", err)
	}

	if got, want := len(commander.commands), 1; got != want {
		t.Fatalf("unexpected command count: got %d want %d", got, want)
	}
	if got, want := commander.commands[0].Name, "bash"; got != want {
		t.Fatalf("unexpected command name: got %q want %q", got, want)
	}
	if got, want := commander.commands[0].Args[0], filepath.Join("/tmp/out", "deploy/azure/deploy.sh"); got != want {
		t.Fatalf("unexpected command arg: got %q want %q", got, want)
	}
	if result.Provider != "azure" {
		t.Fatalf("unexpected provider in result: %q", result.Provider)
	}
}

func TestExecuteBundleFailsForUnsupportedProvider(t *testing.T) {
	exec := &Executor{Commander: &fakeCommander{}}

	if _, err := exec.ExecuteBundle(context.Background(), "/tmp/out", &deployartifact.Bundle{Provider: "bitbucket"}); err == nil {
		t.Fatal("expected unsupported provider to fail")
	}
}

func TestExecuteBundleReturnsRunnerError(t *testing.T) {
	commander := &fakeCommander{err: errors.New("boom")}
	exec := &Executor{Commander: commander}

	if _, err := exec.ExecuteBundle(context.Background(), "/tmp/out", &deployartifact.Bundle{Provider: "azure"}); err == nil {
		t.Fatal("expected runner error to propagate")
	}
}

func TestExecuteAndRecordSavesRelease(t *testing.T) {
	root := t.TempDir()
	commander := &fakeCommander{}
	exec := &Executor{Commander: commander}

	result, err := exec.ExecuteAndRecord(context.Background(), root, "/tmp/out", &deployartifact.Bundle{
		Provider: "azure",
		Files:    map[string]string{"deploy/azure/deploy.sh": "echo ok"},
	})
	if err != nil {
		t.Fatalf("ExecuteAndRecord returned error: %v", err)
	}
	if result.Release == nil || result.Release.Status != "succeeded" {
		t.Fatal("expected successful recorded release")
	}
	if got := len(result.Release.Events); got < 3 {
		t.Fatalf("expected execution events to be recorded, got %d", got)
	}
	if _, err := LoadCurrentRelease(root); err != nil {
		t.Fatalf("LoadCurrentRelease returned error: %v", err)
	}
}

func TestRollbackToReleaseMaterializesBundleAndRecordsRollback(t *testing.T) {
	root := t.TempDir()
	commander := &fakeCommander{}
	exec := &Executor{Commander: commander}
	target := &ReleaseRecord{
		ID:         "20260423T000001Z",
		Provider:   "azure",
		BundleRoot: "/tmp/original",
		Status:     "succeeded",
		BundleFiles: map[string]string{
			"deploy/azure/deploy.sh": "echo rollback",
		},
	}

	result, err := exec.RollbackToRelease(context.Background(), root, target)
	if err != nil {
		t.Fatalf("RollbackToRelease returned error: %v", err)
	}
	if got, want := len(commander.commands), 1; got != want {
		t.Fatalf("unexpected command count: got %d want %d", got, want)
	}
	if result.Release == nil {
		t.Fatal("expected rollback release metadata")
	}
	if got, want := result.Release.Operation, "rollback"; got != want {
		t.Fatalf("unexpected operation: got %q want %q", got, want)
	}
	if got, want := result.Release.RollbackTargetID, target.ID; got != want {
		t.Fatalf("unexpected rollback target: got %q want %q", got, want)
	}
	if len(result.Release.Events) == 0 {
		t.Fatal("expected rollback execution events to be recorded")
	}
	if _, err := os.Stat(filepath.Join(root, StateDirName, BundlesDirName, target.ID, "deploy/azure/deploy.sh")); err != nil {
		t.Fatalf("expected materialized bundle file: %v", err)
	}
}
