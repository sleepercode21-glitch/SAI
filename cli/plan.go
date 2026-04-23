package cli

import (
	"flag"
	"fmt"

	"github.com/sleepercode/sai/compiler"
	"github.com/sleepercode/sai/utils"
)

type planCommand struct{}

func NewPlanCommand() Command {
	return &planCommand{}
}

func (c *planCommand) Name() string {
	return "plan"
}

func (c *planCommand) Description() string {
	return "Compile a manifest and show the deterministic deployment profile"
}

func (c *planCommand) Run(args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	path := fs.String("path", "sai.sai", "Path to the .sai manifest")
	jsonOutput := fs.Bool("json", false, "Emit JSON output")
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

	if *jsonOutput {
		return printJSON(result)
	}

	fmt.Printf("app=%s profile=%s infra=%s min=%d max=%d estimated=$%d\n",
		result.IR.Application.Name,
		result.Plan.Profile,
		result.Plan.InfraClass,
		result.Plan.MinInstances,
		result.Plan.MaxInstances,
		result.Plan.EstimatedUSD,
	)
	return nil
}
