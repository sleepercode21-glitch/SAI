package cli

import (
	"context"
	"flag"
	"fmt"

	"github.com/sleepercode/sai/executor"
)

type rollbackCommand struct{}

func NewRollbackCommand() Command {
	return &rollbackCommand{}
}

func (c *rollbackCommand) Name() string {
	return "rollback"
}

func (c *rollbackCommand) Description() string {
	return "Reserved for incident rollback execution"
}

func (c *rollbackCommand) Run(args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	releaseID := fs.String("release", "", "Release identifier")
	stateRoot := fs.String("state-root", ".sai-out", "Root directory containing SAI execution state")
	execute := fs.Bool("execute", false, "Execute the rollback target after resolving it")
	preflight := fs.Bool("preflight", true, "Run provider tool preflight checks before rollback execution")
	if err := fs.Parse(args); err != nil {
		return err
	}
	record, err := resolveRelease(*stateRoot, *releaseID)
	if err != nil {
		return err
	}
	fmt.Printf("rollback target=%s provider=%s bundle=%s\n", record.ID, record.Provider, record.BundleRoot)
	if !*execute {
		fmt.Println("rerun with --execute to materialize the stored bundle and restore this release")
		return nil
	}

	bundle := executor.BundleFromRelease(record)
	if *preflight {
		checks, err := executor.Preflight(context.Background(), bundle)
		if err != nil {
			return err
		}
		for _, check := range checks {
			fmt.Printf("preflight %s: %s (%s)\n", check.Name, check.Status, check.Detail)
			if check.Status != "ok" {
				return fmt.Errorf("preflight failed for %s", check.Name)
			}
		}
	}

	result, err := executor.New().RollbackToRelease(context.Background(), *stateRoot, record)
	if err != nil {
		return err
	}
	fmt.Printf("rollback executed %d command(s) for %s\n", len(result.Commands), result.Provider)
	if result.Release != nil {
		fmt.Printf("release=%s rollback_of=%s log=%s\n", result.Release.ID, record.ID, result.Release.LogPath)
	}
	return nil
}
