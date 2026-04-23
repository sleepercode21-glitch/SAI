package cli

import (
	"errors"
	"flag"
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
	_ = fs.String("release", "", "Release identifier")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return errors.New("rollback is not implemented yet; the current slice stops after planning")
}
