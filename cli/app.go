package cli

import (
	"errors"
	"fmt"
)

type Command interface {
	Name() string
	Description() string
	Run(args []string) error
}

type App struct {
	commands map[string]Command
}

func NewApp() *App {
	app := &App{commands: map[string]Command{}}
	for _, command := range []Command{
		NewInitCommand(),
		NewValidateCommand(),
		NewPlanCommand(),
		NewDeployCommand(),
		NewLogsCommand(),
		NewRollbackCommand(),
	} {
		app.commands[command.Name()] = command
	}
	return app
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		return a.help()
	}

	name := args[0]
	command, ok := a.commands[name]
	if !ok {
		return fmt.Errorf("unknown command %q", name)
	}
	return command.Run(args[1:])
}

func (a *App) help() error {
	fmt.Println("sai compiles backend intent into deterministic deployment artifacts.")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  sai <command> [flags]")
	fmt.Println("")
	fmt.Println("Commands:")
	for _, name := range []string{"init", "validate", "plan", "deploy", "logs", "rollback"} {
		command := a.commands[name]
		fmt.Printf("  %-10s %s\n", command.Name(), command.Description())
	}
	return errors.New("no command provided")
}
