package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sleepercode/sai/compiler"
	deployartifact "github.com/sleepercode/sai/compiler/deploy"
	"github.com/sleepercode/sai/executor"
	"github.com/sleepercode/sai/utils"
)

type deployCommand struct{}

func NewDeployCommand() Command {
	return &deployCommand{}
}

func (c *deployCommand) Name() string {
	return "deploy"
}

func (c *deployCommand) Description() string {
	return "Reserved for the deploy execution pipeline"
}

func (c *deployCommand) Run(args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	path := fs.String("path", "sai.sai", "Path to the .sai manifest")
	outputDir := fs.String("output-dir", ".sai-out", "Directory to write deployment bundle files")
	execute := fs.Bool("execute", false, "Execute the generated deployment bundle after writing it")
	preflight := fs.Bool("preflight", true, "Run provider tool preflight checks before execution")
	if err := fs.Parse(args); err != nil {
		return err
	}

	manifestPath, err := utils.ResolveManifestPath(*path)
	if err != nil {
		return err
	}
	result, err := compiler.PlanFile(manifestPath)
	if err != nil {
		return err
	}

	if err := writeBundle(*outputDir, result.DeployBundle); err != nil {
		return err
	}
	fmt.Printf("wrote deploy bundle to %s\n", *outputDir)
	for _, file := range deployartifact.BundlePaths(result.DeployBundle) {
		fmt.Printf("%s\n", filepath.Join(*outputDir, file))
	}

	if *execute {
		exec := executor.New()
		if *preflight {
			checks, err := executor.Preflight(context.Background(), result.DeployBundle)
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
		executionResult, err := exec.ExecuteAndRecord(context.Background(), *outputDir, *outputDir, result.DeployBundle)
		if err != nil {
			return err
		}
		fmt.Printf("executed %d deployment command(s) for %s\n", len(executionResult.Commands), executionResult.Provider)
		if executionResult.Release != nil {
			fmt.Printf("release=%s log=%s\n", executionResult.Release.ID, executionResult.Release.LogPath)
		}
	}
	return nil
}

func writeBundle(root string, bundle *deployartifact.Bundle) error {
	for _, relativePath := range deployartifact.BundlePaths(bundle) {
		absolutePath := filepath.Join(root, relativePath)
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(absolutePath, []byte(bundle.Files[relativePath]), 0o644); err != nil {
			return err
		}
	}
	return nil
}
