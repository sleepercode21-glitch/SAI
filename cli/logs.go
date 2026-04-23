package cli

import (
	"errors"
	"flag"
)

type logsCommand struct{}

func NewLogsCommand() Command {
	return &logsCommand{}
}

func (c *logsCommand) Name() string {
	return "logs"
}

func (c *logsCommand) Description() string {
	return "Reserved for deployment and runtime log streaming"
}

func (c *logsCommand) Run(args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	_ = fs.Bool("follow", false, "Follow log output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return errors.New("logs is not implemented yet; the current slice stops after planning")
}
