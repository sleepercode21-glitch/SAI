package cli

import (
	"flag"
	"fmt"

	"github.com/sleepercode/sai/compiler"
	"github.com/sleepercode/sai/utils"
)

type validateCommand struct{}

func NewValidateCommand() Command {
	return &validateCommand{}
}

func (c *validateCommand) Name() string {
	return "validate"
}

func (c *validateCommand) Description() string {
	return "Parse and normalize a .sai manifest"
}

func (c *validateCommand) Run(args []string) error {
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

	fmt.Printf("manifest %s is valid\n", manifestPath)
	fmt.Printf("app=%s service=%s runtime=%s exposure=%s\n",
		result.IR.Application.Name,
		result.IR.Service.Name,
		result.IR.Service.Runtime,
		result.IR.Service.Exposure,
	)
	return nil
}
