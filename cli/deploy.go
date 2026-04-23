package cli

import (
	"errors"
	"flag"
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
	_ = fs.String("path", "sai.sai", "Path to the .sai manifest")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return errors.New("deploy is not implemented yet; the current slice stops after planning")
}
